package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
	"zori/internal/natsstream"
	"zori/internal/storage/clickhouse"
	clickhouseModels "zori/internal/storage/clickhouse/models"
	"zori/services/payments/types"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	paymentStream  = "payments:raw"
	paymentSubject = "payments:raw"
)

type PaymentProcessor struct {
	natsStream *natsstream.Stream
	clickDb    *clickhouse.ClickhouseDB

	consumerJsConn jetstream.JetStream
	consumer       jetstream.Consumer

	cancelConsumer context.CancelFunc
	ctx            context.Context
}

func NewPaymentProcessor(natsStream *natsstream.Stream, clickDb *clickhouse.ClickhouseDB) *PaymentProcessor {
	err := natsStream.UpsertJetStream(paymentStream, paymentSubject)
	if err != nil {
		panic(err)
	}

	p := &PaymentProcessor{
		natsStream: natsStream,
		clickDb:    clickDb,
	}

	p.ctx, p.cancelConsumer = context.WithCancel(context.Background())

	jsConn, err := jetstream.New(p.natsStream.GetConnection())
	if err != nil {
		panic(err)
	}

	p.consumerJsConn = jsConn

	if consumer, err := p.consumerJsConn.Consumer(p.ctx, paymentStream, "payment-processor"); err != nil {
		if errors.Is(err, jetstream.ErrConsumerNotFound) {
			if p.consumer, err = p.consumerJsConn.CreateConsumer(p.ctx, paymentStream, jetstream.ConsumerConfig{
				Name:    "payment-processor",
				Durable: "payment-processor",
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

func (p *PaymentProcessor) Start() error {
	_, err := p.consumer.Consume(func(msg jetstream.Msg) {
		var paymentFrame types.PaymentEventFrame
		if err := json.Unmarshal(msg.Data(), &paymentFrame); err != nil {
			log.Printf("Failed to unmarshal payment event: %v", err)
			msg.Nak()
			return
		}

		if err := p.processPayment(&paymentFrame); err != nil {
			log.Printf("Failed to process payment: %v", err)
			msg.Nak()
			return
		}

		msg.Ack()
	})

	return err
}

func (p *PaymentProcessor) Stop() error {
	p.cancelConsumer()
	p.consumerJsConn.Conn().Close()
	return nil
}

func (p *PaymentProcessor) processPayment(frame *types.PaymentEventFrame) error {
	if frame.PaymentID == "" {
		return fmt.Errorf("payment ID is required")
	}

	if frame.Amount <= 0 {
		return nil
	}

	var status clickhouseModels.PaymentStatus
	switch frame.PaymentStatus {
	case "succeeded":
		status = clickhouseModels.PaymentStatusSucceeded
	case "failed":
		status = clickhouseModels.PaymentStatusFailed
	case "pending":
		status = clickhouseModels.PaymentStatusPending
	case "refunded":
		status = clickhouseModels.PaymentStatusRefunded
	default:
		status = clickhouseModels.PaymentStatusSucceeded
	}

	if err := p.clickDb.Ping(context.Background()); err != nil {
		return fmt.Errorf("clickhouse ping failed: %w", err)
	}

	err := p.clickDb.Db().AsyncInsert(context.Background(),
		`INSERT INTO payment_events (
			payment_id, visitor_id, provider_type, payment_status, product_name,
			amount, currency, payment_timestamp_utc, server_timestamp_utc,
			project_id, organization_id, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, true,
		frame.PaymentID,
		frame.VisitorID,
		frame.ProviderType,
		status,
		frame.ProductName,
		frame.Amount,
		frame.Currency,
		frame.PaymentTimestamp,
		time.Now().UTC(),
		frame.ProjectID,
		frame.OrganizationID,
		frame.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to insert payment event: %w", err)
	}

	log.Printf("Successfully processed payment %s (visitor: %v, amount: %d %s)",
		frame.PaymentID,
		frame.VisitorID,
		frame.Amount,
		frame.Currency,
	)

	return nil
}
