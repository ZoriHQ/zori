package services

import (
	"context"
	"sync"
	"time"
	"zori/internal/metrics"
	"zori/internal/storage/clickhouse"
	"zori/services/ingestion/types"

	trieclassifier "github.com/ZoriHQ/trie-url-classifier"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	defaultBatchSize = 200
)

type BatchItem struct {
	EventFrame *types.ClientEventFrameV1
	Msg        jetstream.Msg
}

type BatchProcessor struct {
	clickDb     *clickhouse.ClickhouseDB
	natsMetrics *metrics.NatsMetrics

	batchSize int
	batch     []BatchItem
	batchMu   sync.Mutex

	eventChan chan BatchItem
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

func NewBatchProcessor(clickDb *clickhouse.ClickhouseDB, natsMetrics *metrics.NatsMetrics) *BatchProcessor {
	return &BatchProcessor{
		clickDb:     clickDb,
		natsMetrics: natsMetrics,
		batchSize:   defaultBatchSize,
		batch:       make([]BatchItem, 0, defaultBatchSize),
		eventChan:   make(chan BatchItem, 100),
		stopChan:    make(chan struct{}),
	}
}

func (bp *BatchProcessor) Start() error {
	bp.wg.Add(1)
	go bp.processBatches()
	return nil
}

func (bp *BatchProcessor) Stop() error {
	close(bp.stopChan)
	bp.wg.Wait()

	bp.batchMu.Lock()
	if len(bp.batch) > 0 {
		bp.processBatch(bp.batch)
	}
	bp.batchMu.Unlock()

	return nil
}

func (bp *BatchProcessor) AddEvent(eventFrame *types.ClientEventFrameV1, msg jetstream.Msg) {
	bp.eventChan <- BatchItem{
		EventFrame: eventFrame,
		Msg:        msg,
	}
}

func (bp *BatchProcessor) processBatches() {
	defer bp.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bp.stopChan:
			return
		case item := <-bp.eventChan:
			bp.batchMu.Lock()
			bp.batch = append(bp.batch, item)
			shouldProcess := len(bp.batch) >= bp.batchSize
			bp.batchMu.Unlock()

			if shouldProcess {
				bp.batchMu.Lock()
				batchToProcess := bp.batch
				bp.batch = make([]BatchItem, 0, bp.batchSize)
				bp.batchMu.Unlock()

				bp.processBatch(batchToProcess)
			}
		case <-ticker.C:
			bp.batchMu.Lock()
			if len(bp.batch) > 0 {
				batchToProcess := bp.batch
				bp.batch = make([]BatchItem, 0, bp.batchSize)
				bp.batchMu.Unlock()

				bp.processBatch(batchToProcess)
			} else {
				bp.batchMu.Unlock()
			}
		}
	}
}

func (bp *BatchProcessor) processBatch(batch []BatchItem) {
	if len(batch) == 0 {
		return
	}

	startTime := time.Now()

	urls := make([]string, 0, len(batch))
	for _, item := range batch {
		if item.EventFrame.PageURL != "" {
			urls = append(urls, item.EventFrame.PageURL)
		}
	}

	classifier := trieclassifier.NewClassifier()
	if len(urls) > 0 {
		classifier.Learn(urls)
	}

	for i := range batch {
		if batch[i].EventFrame.PageURL != "" {
			pattern := classifier.Classify(batch[i].EventFrame.PageURL)
			batch[i].EventFrame.PagePattern = &pattern
		}
	}

	ctx := context.Background()

	if err := bp.clickDb.Ping(ctx); err != nil {
		bp.natsMetrics.RecordClickHouseError("events", "connection_error")
		for _, item := range batch {
			bp.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
			item.Msg.Nak()
		}
		return
	}

	insertStart := time.Now()

	batchInsert, err := bp.clickDb.Db().PrepareBatch(ctx,
		`INSERT INTO events (
			ip, visitor_id, session_id, browser_name, os_name, device_type, client_generated_event_id, event_name,
			location_country_iso, location_city, location_latitude, location_longitude,
			client_timestamp_utc, server_timestamp_utc, user_agent, host,
			page_url, page_path, page_pattern, referrer_url, referrer_domain, referrer_path, utm_parameters,
			click_element_tag, click_element_selector, click_element_text,
			click_position_x, click_position_y, click_screen_width, click_screen_height,
			click_element_type, click_element_category, is_cta_click, link_destination, is_external_link, is_download_link,
			project_id, organization_id,
			user_id, external_id, email_hash
		)`)

	if err != nil {
		bp.natsMetrics.RecordClickHouseError("events", "prepare_error")
		for _, item := range batch {
			bp.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
			item.Msg.Nak()
		}
		return
	}

	for _, item := range batch {
		eventFrame := item.EventFrame

		var (
			clickPositionX       *float64
			clickPositionY       *float64
			clickScreenWidth     *uint16
			clickScreenHeight    *uint16
			clickElementTag      *string
			clickElementSelector *string
			clickElementText     *string
		)

		if eventFrame.ClickPosition != nil {
			clickPositionX = &eventFrame.ClickPosition.X
			clickPositionY = &eventFrame.ClickPosition.Y
			clickScreenWidth = &eventFrame.ClickPosition.ScreenWidth
			clickScreenHeight = &eventFrame.ClickPosition.ScreenHeight
		}

		if eventFrame.ClickElement != nil {
			clickElementTag = &eventFrame.ClickElement.Tag
			clickElementSelector = &eventFrame.ClickElement.Selector
			clickElementText = &eventFrame.ClickElement.Text
		}

		err = batchInsert.Append(
			eventFrame.IP,
			eventFrame.VisitorID,
			eventFrame.SessionID,
			eventFrame.BrowserName,
			eventFrame.OsName,
			eventFrame.DeviceType,
			eventFrame.ClientGeneratedEventID,
			eventFrame.EventName,
			eventFrame.LocationCountryISO,
			eventFrame.LocationCity,
			eventFrame.LocationLatitude,
			eventFrame.LocationLongitude,
			eventFrame.ClientTimeStampUTC,
			time.Now().UTC(),
			eventFrame.UserAgent,
			eventFrame.Host,
			eventFrame.PageURL,
			eventFrame.PagePath,
			eventFrame.PagePattern,
			eventFrame.Referrer,
			eventFrame.ReferredDomain,
			eventFrame.ReferrerPath,
			eventFrame.UTMParameters,
			clickElementTag,
			clickElementSelector,
			clickElementText,
			clickPositionX,
			clickPositionY,
			clickScreenWidth,
			clickScreenHeight,
			eventFrame.ClickElementType,
			eventFrame.ClickElementCategory,
			eventFrame.IsCTAClick,
			eventFrame.LinkDestination,
			eventFrame.IsExternalLink,
			eventFrame.IsDownloadLink,
			eventFrame.ProjectID,
			eventFrame.OrganizationID,
			eventFrame.UserID,
			eventFrame.ExternalID,
			eventFrame.EmailHash,
		)

		if err != nil {
			bp.natsMetrics.RecordClickHouseError("events", "append_error")
			for _, item := range batch {
				bp.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
				item.Msg.Nak()
			}
			return
		}
	}

	if err := batchInsert.Send(); err != nil {
		bp.natsMetrics.RecordClickHouseError("events", "insert_error")
		for _, item := range batch {
			bp.natsMetrics.RecordMessageProcessed(rawEventsStream, "event-enricher", "clickhouse_error", time.Since(startTime))
			bp.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
			item.Msg.Nak()
		}
		return
	}

	bp.natsMetrics.RecordClickHouseInsert("events", time.Since(insertStart))

	for _, item := range batch {
		bp.natsMetrics.RecordMessageProcessed(rawEventsStream, "event-enricher", "success", time.Since(startTime))
		bp.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "ack")
		item.Msg.Ack()
	}
}
