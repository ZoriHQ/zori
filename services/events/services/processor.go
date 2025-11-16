package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"zori/internal/logger"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/storage/clickhouse"
	"zori/internal/telemetry"
	"zori/services/ingestion/types"

	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/trace"
)

const (
	rawEventsStream  = "events:raw"
	rawEventsSubject = "events:raw"
)

type Processor struct {
	natsStream *natsstream.Stream

	consumerJsConnn jetstream.JetStream
	consumer        jetstream.Consumer

	cancelConsumer context.CancelFunc
	ctx            context.Context

	clickDb *clickhouse.ClickhouseDB

	stages         []ProcessorStage
	natsMetrics    *metrics.NatsMetrics
	tracerProvider *telemetry.Provider
	tracer         trace.Tracer
	logger         *logger.Logger
}

func NewProcessor(
	natsStream *natsstream.Stream,
	clickDb *clickhouse.ClickhouseDB,
	natsMetrics *metrics.NatsMetrics,
	tracerProvider *telemetry.Provider,
	log *logger.Logger,
) *Processor {
	err := natsStream.UpsertJetStream(rawEventsStream, rawEventsSubject)
	if err != nil {
		panic(err)
	}

	processingStages := []ProcessorStage{
		NewStageLocation(),
		NewStagePage(),
		NewStageUserAgent(),
		NewStageReferrer(),
		NewStageIdentity(),
		NewStageClickClassification(),
	}

	var tracer trace.Tracer
	if tracerProvider != nil {
		tracer = tracerProvider.Tracer("zori.processor")
	}

	p := &Processor{
		natsStream:     natsStream,
		clickDb:        clickDb,
		stages:         processingStages,
		natsMetrics:    natsMetrics,
		tracerProvider: tracerProvider,
		tracer:         tracer,
		logger:         log,
	}

	p.ctx, p.cancelConsumer = context.WithCancel(context.Background())

	jsConn, err := jetstream.New(p.natsStream.GetConnection())
	if err != nil {
		panic(err)
	}

	p.consumerJsConnn = jsConn
	if consumer, err := p.consumerJsConnn.Consumer(p.ctx, rawEventsStream, "event-enricher"); err != nil {
		if errors.Is(err, jetstream.ErrConsumerNotFound) {
			if p.consumer, err = p.consumerJsConnn.CreateConsumer(p.ctx, rawEventsStream, jetstream.ConsumerConfig{
				Name:    "event-enricher",
				Durable: "event-enricher",
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

func (p *Processor) Start() error {
	_, err := p.consumer.Consume(func(msg jetstream.Msg) {
		startTime := time.Now()

		ctx, span := p.startProcessingSpan(msg)
		defer span.End()

		var eventFrame types.ClientEventFrameV1
		if err := json.Unmarshal(msg.Data(), &eventFrame); err != nil {
			telemetry.RecordError(span, err)
			p.logger.WithContext(ctx).Error("Failed to unmarshal event frame", "error", err)
			p.natsMetrics.RecordMessageProcessed(rawEventsStream, "event-enricher", "unmarshal_error", time.Since(startTime))
			p.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
			msg.Nak()
			return
		}

		telemetry.AddIngestionAttributes(span, eventFrame.OrganizationID, eventFrame.ProjectID, eventFrame.VisitorID, "process")

		if err := p.processEvent(ctx, &eventFrame); err != nil {
			telemetry.RecordError(span, err)
			p.logger.WithContext(ctx).Error("Failed to process event",
				"error", err,
				"org_id", eventFrame.OrganizationID,
				"project_id", eventFrame.ProjectID)
			p.natsMetrics.RecordMessageProcessed(rawEventsStream, "event-enricher", "process_error", time.Since(startTime))
			p.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
			msg.Nak()
			return
		}

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

		if err := p.clickDb.Ping(ctx); err != nil {
			p.logger.WithContext(ctx).Error("Error pinging database", "error", err)
		}

		insertStart := time.Now()
		if err := p.clickDb.Db().AsyncInsert(context.Background(),
			`INSERT INTO events (
				ip, visitor_id, session_id, browser_name, os_name, device_type, client_generated_event_id, event_name,
				location_country_iso, location_city, location_latitude, location_longitude,
				client_timestamp_utc, server_timestamp_utc, user_agent, host,
				page_url, page_path, referrer_url, referrer_domain, referrer_path, utm_parameters,
				click_element_tag, click_element_selector, click_element_text,
				click_position_x, click_position_y, click_screen_width, click_screen_height,
				click_element_type, click_element_category, is_cta_click, link_destination, is_external_link, is_download_link,
				project_id, organization_id,
				user_id, external_id, email_hash
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, true,
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
		); err != nil {
			p.natsMetrics.RecordClickHouseError("events", "insert_error")
			p.natsMetrics.RecordMessageProcessed(rawEventsStream, "event-enricher", "clickhouse_error", time.Since(startTime))
			p.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
			msg.Nak()
			return
		}
		p.natsMetrics.RecordClickHouseInsert("events", time.Since(insertStart))

		p.natsMetrics.RecordMessageProcessed(rawEventsStream, "event-enricher", "success", time.Since(startTime))
		p.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "ack")
		msg.Ack()

		marshalEventFrame, err := json.Marshal(eventFrame)
		if err != nil {
			p.logger.WithContext(ctx).Error("Error marshaling event frame", "error", err)
		} else {
			p.natsStream.GetConnection().Publish(fmt.Sprintf("events:project:%s", eventFrame.ProjectID), marshalEventFrame)
		}

		telemetry.SetStatus(span, nil)
		p.logger.WithContext(ctx).Debug("Event processed successfully",
			"org_id", eventFrame.OrganizationID,
			"project_id", eventFrame.ProjectID,
			"visitor_id", eventFrame.VisitorID)
	})

	return err
}

func (p *Processor) Stop() error {
	p.cancelConsumer()
	p.consumerJsConnn.Conn().Close()
	return nil
}

func (p *Processor) processEvent(ctx context.Context, eventFrame *types.ClientEventFrameV1) error {
	stageNames := []string{
		"location",
		"page",
		"user_agent",
		"referrer",
		"identity",
		"click_classification",
	}

	for i, stage := range p.stages {
		stageName := stageNames[i]
		timer := p.natsMetrics.NewStageTimer(stageName)

		if err := stage.ProcessFrame(eventFrame); err != nil {
			timer.Error()
			return err
		}

		timer.Done()
	}

	return nil
}

// startProcessingSpan creates a new span for NATS message processing
func (p *Processor) startProcessingSpan(msg jetstream.Msg) (context.Context, trace.Span) {
	if p.tracer == nil {
		return context.Background(), trace.SpanFromContext(context.Background())
	}

	ctx, span := p.tracer.Start(context.Background(), "processor.process_event",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)

	return ctx, span
}
