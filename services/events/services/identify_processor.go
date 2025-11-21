package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/storage/clickhouse"
	"zori/internal/telemetry"
	"zori/services/ingestion/types"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	identifyEventsStream  = "events:identify"
	identifyEventsSubject = "events:identify"
)

type IdentifyProcessor struct {
	natsStream *natsstream.Stream

	consumerJsConn jetstream.JetStream
	consumer       jetstream.Consumer

	cancelConsumer context.CancelFunc
	ctx            context.Context

	clickDb     *clickhouse.ClickhouseDB
	natsMetrics *metrics.NatsMetrics
	logger      *telemetry.Logger
}

func NewIdentifyProcessor(natsStream *natsstream.Stream, clickDb *clickhouse.ClickhouseDB, natsMetrics *metrics.NatsMetrics, logger *telemetry.Logger) *IdentifyProcessor {
	err := natsStream.UpsertJetStream(identifyEventsStream, identifyEventsSubject)
	if err != nil {
		panic(err)
	}

	p := &IdentifyProcessor{
		natsStream:  natsStream,
		clickDb:     clickDb,
		natsMetrics: natsMetrics,
		logger:      logger,
	}

	p.ctx, p.cancelConsumer = context.WithCancel(context.Background())

	jsConn, err := jetstream.New(p.natsStream.GetConnection())
	if err != nil {
		panic(err)
	}

	p.consumerJsConn = jsConn
	if consumer, err := p.consumerJsConn.Consumer(p.ctx, identifyEventsStream, "identify-processor"); err != nil {
		if errors.Is(err, jetstream.ErrConsumerNotFound) {
			if p.consumer, err = p.consumerJsConn.CreateConsumer(p.ctx, identifyEventsStream, jetstream.ConsumerConfig{
				Name:    "identify-processor",
				Durable: "identify-processor",
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

func (p *IdentifyProcessor) Start() error {
	_, err := p.consumer.Consume(func(msg jetstream.Msg) {
		startTime := time.Now()

		var identifyFrame types.IdentifyEventFrameV1
		if err := json.Unmarshal(msg.Data(), &identifyFrame); err != nil {
			p.logger.Error("Failed to unmarshal identify event", telemetry.Error(err))
			p.natsMetrics.RecordMessageProcessed(identifyEventsStream, "identify-processor", "unmarshal_error", time.Since(startTime))
			p.natsMetrics.RecordMessageAck(identifyEventsStream, "identify-processor", "nak")
			if err := msg.Nak(); err != nil {
				p.logger.Error("Failed to nak message after unmarshal error", telemetry.Error(err))
			}
			return
		}

		if err := p.updateVisitorEvents(&identifyFrame); err != nil {
			p.logger.Error("Failed to update visitor events", telemetry.Error(err), telemetry.String("visitor_id", identifyFrame.VisitorID))
			p.natsMetrics.RecordMessageProcessed(identifyEventsStream, "identify-processor", "update_error", time.Since(startTime))
			p.natsMetrics.RecordMessageAck(identifyEventsStream, "identify-processor", "nak")
			if err := msg.Nak(); err != nil {
				p.logger.Error("Failed to nak message after update error", telemetry.Error(err))
			}
			return
		}

		p.natsMetrics.RecordMessageProcessed(identifyEventsStream, "identify-processor", "success", time.Since(startTime))
		p.natsMetrics.RecordMessageAck(identifyEventsStream, "identify-processor", "ack")
		if err := msg.Ack(); err != nil {
			p.logger.Error("Failed to ack message after successful processing", telemetry.Error(err))
		}
	})

	return err
}

func (p *IdentifyProcessor) Stop() error {
	p.cancelConsumer()
	p.consumerJsConn.Conn().Close()
	return nil
}

func (p *IdentifyProcessor) updateVisitorEvents(identifyFrame *types.IdentifyEventFrameV1) error {
	var (
		userID     *string
		externalID *string
		emailHash  *string
	)

	userID = identifyFrame.AppID

	if identifyFrame.Email != nil && *identifyFrame.Email != "" {
		hash := hashEmailSHA256(*identifyFrame.Email)
		emailHash = &hash
	}

	query := `
		ALTER TABLE events
		UPDATE
			user_id = ?,
			external_id = ?,
			email_hash = ?
		WHERE
			visitor_id = ? AND
			project_id = ? AND
			organization_id = ?
	`

	err := p.clickDb.Db().Exec(
		context.Background(),
		query,
		userID,
		externalID,
		emailHash,
		identifyFrame.VisitorID,
		identifyFrame.ProjectID,
		identifyFrame.OrganizationID,
	)

	if err != nil {
		return fmt.Errorf("failed to update visitor events: %w", err)
	}

	p.logger.Info("Updated events with identity information", telemetry.String("visitor_id", identifyFrame.VisitorID), telemetry.String("project_id", identifyFrame.ProjectID))
	return nil
}

func hashEmailSHA256(email string) string {
	hasher := sha256.New()
	hasher.Write([]byte(email))
	return hex.EncodeToString(hasher.Sum(nil))
}
