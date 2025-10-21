package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"zori/internal/natsstream"
	"zori/internal/storage/clickhouse"
	"zori/services/ingestion/types"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	identifyEventsStream  = "events:identify"
	identifyEventsSubject = "events:identify"
)

// IdentifyProcessor handles identify events and updates ClickHouse events with identity information
type IdentifyProcessor struct {
	natsStream *natsstream.Stream

	consumerJsConn jetstream.JetStream
	consumer       jetstream.Consumer

	cancelConsumer context.CancelFunc
	ctx            context.Context

	clickDb *clickhouse.ClickhouseDB
}

func NewIdentifyProcessor(natsStream *natsstream.Stream, clickDb *clickhouse.ClickhouseDB) *IdentifyProcessor {
	err := natsStream.UpsertJetStream(identifyEventsStream, identifyEventsSubject)
	if err != nil {
		panic(err)
	}

	p := &IdentifyProcessor{
		natsStream: natsStream,
		clickDb:    clickDb,
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
		var identifyFrame types.IdentifyEventFrameV1
		if err := json.Unmarshal(msg.Data(), &identifyFrame); err != nil {
			log.Printf("Failed to unmarshal identify event: %v", err)
			msg.Nak()
			return
		}

		// Update all events for this visitor with identity information
		if err := p.updateVisitorEvents(&identifyFrame); err != nil {
			log.Printf("Failed to update visitor events: %v", err)
			msg.Nak()
			return
		}

		msg.Ack()
	})

	return err
}

func (p *IdentifyProcessor) Stop() error {
	p.cancelConsumer()
	p.consumerJsConn.Conn().Close()
	return nil
}

// updateVisitorEvents updates all events for a visitor with their identity information
func (p *IdentifyProcessor) updateVisitorEvents(identifyFrame *types.IdentifyEventFrameV1) error {
	var (
		userID     *string
		externalID *string
		emailHash  *string
	)

	// Set identity fields
	userID = identifyFrame.AppID

	// Hash email if provided
	if identifyFrame.Email != nil && *identifyFrame.Email != "" {
		hash := hashEmailSHA256(*identifyFrame.Email)
		emailHash = &hash
	}

	// Update all past events for this visitor
	// Note: ClickHouse doesn't support UPDATE efficiently, so we use ALTER TABLE UPDATE
	// This is an asynchronous operation in ClickHouse
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

	log.Printf("Updated events for visitor %s with identity information", identifyFrame.VisitorID)
	return nil
}

// hashEmailSHA256 creates a SHA256 hash of the email
func hashEmailSHA256(email string) string {
	hasher := sha256.New()
	hasher.Write([]byte(email))
	return hex.EncodeToString(hasher.Sum(nil))
}
