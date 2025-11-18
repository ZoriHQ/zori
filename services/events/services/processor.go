package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/storage/clickhouse"
	"zori/services/ingestion/types"

	"github.com/nats-io/nats.go/jetstream"
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
	batchProcessor *BatchProcessor
}

func NewProcessor(natsStream *natsstream.Stream, clickDb *clickhouse.ClickhouseDB, natsMetrics *metrics.NatsMetrics, batchProcessor *BatchProcessor) *Processor {
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

	p := &Processor{
		natsStream:     natsStream,
		clickDb:        clickDb,
		stages:         processingStages,
		natsMetrics:    natsMetrics,
		batchProcessor: batchProcessor,
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

		var eventFrame types.ClientEventFrameV1
		if err := json.Unmarshal(msg.Data(), &eventFrame); err != nil {
			p.natsMetrics.RecordMessageProcessed(rawEventsStream, "event-enricher", "unmarshal_error", time.Since(startTime))
			p.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
			msg.Nak()
			return
		}

		if err := p.processEvent(&eventFrame); err != nil {
			p.natsMetrics.RecordMessageProcessed(rawEventsStream, "event-enricher", "process_error", time.Since(startTime))
			p.natsMetrics.RecordMessageAck(rawEventsStream, "event-enricher", "nak")
			msg.Nak()
			return
		}

		p.batchProcessor.AddEvent(&eventFrame, msg)

		marshalEventFrame, err := json.Marshal(eventFrame)
		if err != nil {
			log.Printf("Error marshaling event frame: %v", err)
		} else {
			p.natsStream.GetConnection().Publish(fmt.Sprintf("events:project:%s", eventFrame.ProjectID), marshalEventFrame)
		}
	})

	return err
}

func (p *Processor) Stop() error {
	p.cancelConsumer()
	p.consumerJsConnn.Conn().Close()
	return nil
}

func (p *Processor) processEvent(eventFrame *types.ClientEventFrameV1) error {
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
