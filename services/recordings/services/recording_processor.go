package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/storage/clickhouse"
	"zori/internal/telemetry"
	"zori/services/ingestion/types"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	rawRecordingsStream  = "recordings:raw"
	rawRecordingsSubject = "recordings:raw"
)

type RecordingProcessor struct {
	natsStream *natsstream.Stream

	consumerJsConn jetstream.JetStream
	consumer       jetstream.Consumer

	cancelConsumer context.CancelFunc
	ctx            context.Context

	clickDb *clickhouse.ClickhouseDB

	natsMetrics    *metrics.NatsMetrics
	batchProcessor *RecordingBatchProcessor
	logger         *telemetry.Logger
}

func NewRecordingProcessor(natsStream *natsstream.Stream, clickDb *clickhouse.ClickhouseDB, natsMetrics *metrics.NatsMetrics, batchProcessor *RecordingBatchProcessor, logger *telemetry.Logger) *RecordingProcessor {
	err := natsStream.UpsertJetStream(rawRecordingsStream, rawRecordingsSubject)
	if err != nil {
		panic(err)
	}

	p := &RecordingProcessor{
		natsStream:     natsStream,
		clickDb:        clickDb,
		natsMetrics:    natsMetrics,
		batchProcessor: batchProcessor,
		logger:         logger,
	}

	p.ctx, p.cancelConsumer = context.WithCancel(context.Background())

	jsConn, err := jetstream.New(p.natsStream.GetConnection())
	if err != nil {
		panic(err)
	}

	p.consumerJsConn = jsConn
	if consumer, err := p.consumerJsConn.Consumer(p.ctx, rawRecordingsStream, "recording-processor"); err != nil {
		if errors.Is(err, jetstream.ErrConsumerNotFound) {
			if p.consumer, err = p.consumerJsConn.CreateConsumer(p.ctx, rawRecordingsStream, jetstream.ConsumerConfig{
				Name:    "recording-processor",
				Durable: "recording-processor",
			}); err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	} else {
		p.consumer = consumer
	}

	return p
}

func (p *RecordingProcessor) Start() error {
	_, err := p.consumer.Consume(func(msg jetstream.Msg) {
		startTime := time.Now()

		var recordingFrame types.RecordingEventFrameV1
		if err := json.Unmarshal(msg.Data(), &recordingFrame); err != nil {
			p.natsMetrics.RecordMessageProcessed(rawRecordingsStream, "recording-processor", "unmarshal_error", time.Since(startTime))
			p.natsMetrics.RecordMessageAck(rawRecordingsStream, "recording-processor", "nak")
			if err := msg.Nak(); err != nil {
				p.logger.Error("Failed to nak message after unmarshal error", telemetry.Error(err))
			}
			return
		}

		p.batchProcessor.AddRecording(&recordingFrame, msg)
	})

	return err
}

func (p *RecordingProcessor) Stop() error {
	p.cancelConsumer()
	p.consumerJsConn.Conn().Close()
	return nil
}
