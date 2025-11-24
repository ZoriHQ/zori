package services

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
	"zori/internal/metrics"
	"zori/internal/storage/clickhouse"
	"zori/internal/telemetry"
	"zori/services/ingestion/types"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	defaultRecordingBatchSize = 50
)

type RecordingBatchItem struct {
	RecordingFrame *types.RecordingEventFrameV1
	Msg            jetstream.Msg
}

type RecordingBatchProcessor struct {
	clickDb     *clickhouse.ClickhouseDB
	natsMetrics *metrics.NatsMetrics
	logger      *telemetry.Logger

	batchSize int
	batch     []RecordingBatchItem
	batchMu   sync.Mutex

	recordingChan chan RecordingBatchItem
	stopChan      chan struct{}
	wg            sync.WaitGroup
	started       atomic.Bool
	stopOnce      sync.Once
	lifecycleMu   sync.Mutex
}

func NewRecordingBatchProcessor(clickDb *clickhouse.ClickhouseDB, natsMetrics *metrics.NatsMetrics, logger *telemetry.Logger) *RecordingBatchProcessor {
	return &RecordingBatchProcessor{
		clickDb:       clickDb,
		natsMetrics:   natsMetrics,
		logger:        logger,
		batchSize:     defaultRecordingBatchSize,
		batch:         make([]RecordingBatchItem, 0, defaultRecordingBatchSize),
		recordingChan: make(chan RecordingBatchItem, 100),
		stopChan:      make(chan struct{}),
	}
}

func (bp *RecordingBatchProcessor) Start() error {
	bp.lifecycleMu.Lock()
	defer bp.lifecycleMu.Unlock()

	if !bp.started.CompareAndSwap(false, true) {
		return nil
	}
	bp.wg.Add(1)
	go bp.processBatches()
	return nil
}

func (bp *RecordingBatchProcessor) Stop() error {
	bp.lifecycleMu.Lock()
	if !bp.started.Load() {
		bp.lifecycleMu.Unlock()
		return nil
	}
	bp.lifecycleMu.Unlock()

	bp.stopOnce.Do(func() {
		close(bp.stopChan)
		bp.wg.Wait()

		bp.batchMu.Lock()
		if len(bp.batch) > 0 {
			bp.processBatch(bp.batch)
		}
		bp.batchMu.Unlock()
	})

	return nil
}

func (bp *RecordingBatchProcessor) AddRecording(recordingFrame *types.RecordingEventFrameV1, msg jetstream.Msg) {
	bp.recordingChan <- RecordingBatchItem{
		RecordingFrame: recordingFrame,
		Msg:            msg,
	}
}

func (bp *RecordingBatchProcessor) processBatches() {
	defer bp.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bp.stopChan:
			return
		case item := <-bp.recordingChan:
			bp.batchMu.Lock()
			bp.batch = append(bp.batch, item)
			shouldProcess := len(bp.batch) >= bp.batchSize
			bp.batchMu.Unlock()

			if shouldProcess {
				bp.batchMu.Lock()
				batchToProcess := bp.batch
				bp.batch = make([]RecordingBatchItem, 0, bp.batchSize)
				bp.batchMu.Unlock()

				bp.processBatch(batchToProcess)
			}
		case <-ticker.C:
			bp.batchMu.Lock()
			if len(bp.batch) > 0 {
				batchToProcess := bp.batch
				bp.batch = make([]RecordingBatchItem, 0, bp.batchSize)
				bp.batchMu.Unlock()

				bp.processBatch(batchToProcess)
			} else {
				bp.batchMu.Unlock()
			}
		}
	}
}

func (bp *RecordingBatchProcessor) processBatch(batch []RecordingBatchItem) {
	if len(batch) == 0 {
		return
	}

	startTime := time.Now()
	ctx := context.Background()

	if err := bp.clickDb.Ping(ctx); err != nil {
		bp.natsMetrics.RecordClickHouseError("session_recordings", "connection_error")
		for _, item := range batch {
			bp.natsMetrics.RecordMessageAck(rawRecordingsStream, "recording-processor", "nak")
			if err := item.Msg.Nak(); err != nil {
				bp.logger.Error("Failed to nak message after ClickHouse connection error", telemetry.Error(err))
			}
		}
		return
	}

	insertStart := time.Now()

	batchInsert, err := bp.clickDb.Db().PrepareBatch(ctx,
		`INSERT INTO session_recordings (
			visitor_id, session_id,
			project_id, organization_id,
			page_url, host,
			events, event_count,
			chunk_index, is_final_chunk,
			client_timestamp_utc, server_timestamp_utc,
			user_agent, ip
		)`)

	if err != nil {
		bp.natsMetrics.RecordClickHouseError("session_recordings", "prepare_error")
		for _, item := range batch {
			bp.natsMetrics.RecordMessageAck(rawRecordingsStream, "recording-processor", "nak")
			if err := item.Msg.Nak(); err != nil {
				bp.logger.Error("Failed to nak message after ClickHouse prepare error", telemetry.Error(err))
			}
		}
		return
	}

	for _, item := range batch {
		recordingFrame := item.RecordingFrame

		eventsJSON, err := json.Marshal(recordingFrame.RecordingEvents)
		if err != nil {
			bp.logger.Error("Failed to marshal recording events", telemetry.Error(err))
			eventsJSON = []byte("[]")
		}

		var isFinalChunk uint8
		if recordingFrame.IsFinalChunk {
			isFinalChunk = 1
		}

		err = batchInsert.Append(
			recordingFrame.VisitorID,
			recordingFrame.SessionID,
			recordingFrame.ProjectID,
			recordingFrame.OrganizationID,
			recordingFrame.PageURL,
			recordingFrame.Host,
			string(eventsJSON),
			recordingFrame.EventCount,
			recordingFrame.ChunkIndex,
			isFinalChunk,
			recordingFrame.ClientTimestampUTC,
			time.Now().UTC(),
			recordingFrame.UserAgent,
			recordingFrame.IP,
		)

		if err != nil {
			bp.natsMetrics.RecordClickHouseError("session_recordings", "append_error")
			for _, item := range batch {
				bp.natsMetrics.RecordMessageAck(rawRecordingsStream, "recording-processor", "nak")
				if err := item.Msg.Nak(); err != nil {
					bp.logger.Error("Failed to nak message after ClickHouse append error", telemetry.Error(err))
				}
			}
			return
		}
	}

	if err := batchInsert.Send(); err != nil {
		bp.natsMetrics.RecordClickHouseError("session_recordings", "insert_error")
		for _, item := range batch {
			bp.natsMetrics.RecordMessageProcessed(rawRecordingsStream, "recording-processor", "clickhouse_error", time.Since(startTime))
			bp.natsMetrics.RecordMessageAck(rawRecordingsStream, "recording-processor", "nak")
			if err := item.Msg.Nak(); err != nil {
				bp.logger.Error("Failed to nak message after ClickHouse insert error", telemetry.Error(err))
			}
		}
		return
	}

	bp.natsMetrics.RecordClickHouseInsert("session_recordings", time.Since(insertStart))

	for _, item := range batch {
		bp.natsMetrics.RecordMessageProcessed(rawRecordingsStream, "recording-processor", "success", time.Since(startTime))
		bp.natsMetrics.RecordMessageAck(rawRecordingsStream, "recording-processor", "ack")
		if err := item.Msg.Ack(); err != nil {
			bp.logger.Error("Failed to ack message after ClickHouse insert success", telemetry.Error(err))
		}
	}
}
