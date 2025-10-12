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

	stages []ProcessorStage
}

func NewProcessor(natsStream *natsstream.Stream, clickDb *clickhouse.ClickhouseDB) *Processor {
	err := natsStream.UpsertJetStream(rawEventsStream, rawEventsSubject)
	if err != nil {
		panic(err)
	}

	processingStages := []ProcessorStage{
		NewStageLocation(),
		NewStagePage(),
		NewStageUserAgent(),
		NewStageReferrer(),
	}

	p := &Processor{
		natsStream: natsStream,
		clickDb:    clickDb,
		stages:     processingStages,
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
		var eventFrame types.ClientEventFrameV1
		if err := json.Unmarshal(msg.Data(), &eventFrame); err != nil {
			msg.Nak()
			return
		}

		if err := p.processEvent(&eventFrame); err != nil {
			fmt.Println("Failed to process event", err)
			msg.Nak()
			return
		}

		var (
			clickPositionX *float64
			clickPositionY *float64
		)
		if eventFrame.ClickPosition != nil && len(*eventFrame.ClickPosition) > 0 {
			pos := *eventFrame.ClickPosition
			clickPositionX = &pos[0]
			clickPositionY = &pos[1]
		}

		// eventModelClick := models.Event{
		// 	IP:                     eventFrame.IP,
		// 	VisitorID:              eventFrame.VisitorID,
		// 	ClientGeneratedEventID: eventFrame.ClientGeneratedEventID,
		// 	EventName:              eventFrame.EventName,
		// 	LocationCountryISO:     eventFrame.LocationCountryISO,
		// 	LocationCity:           eventFrame.LocationCity,

		// 	ClientTimestampUTC: eventFrame.ClientTimeStampUTC,
		// 	ServerTimestampUTC: time.Now().UTC(),

		// 	UserAgent:     eventFrame.UserAgent,
		// 	Referrer:      eventFrame.Referrer,
		// 	UTMParameters: eventFrame.UTMParameters,

		// 	ClickOn:        eventFrame.ClickOn,
		// 	ClickPositionX: clickPositionX,
		// 	ClickPositionY: clickPositionY,
		// 	ProjectID:      eventFrame.ProjectID,
		// 	OrganizationID: eventFrame.OrganizationID,
		// }

		if err := p.clickDb.Ping(context.Background()); err != nil {
			fmt.Println(err)
			log.Printf("Error pinging database: %v", err)
		}

		if err := p.clickDb.Db().AsyncInsert(context.Background(),
			`INSERT INTO events (
				ip, visitor_id, browser_name, os_name, device_type, client_generated_event_id, event_name, location_country_iso, location_city, client_timestamp_utc,
				server_timestamp_utc, user_agent, page_url, page_path, referrer_url, referrer_domain, referrer_path, utm_parameters, click_on, click_position_x, click_position_y, project_id,
				organization_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, true,
			eventFrame.IP,
			eventFrame.VisitorID,
			eventFrame.BrowserName,
			eventFrame.OsName,
			eventFrame.DeviceType,
			eventFrame.ClientGeneratedEventID,
			eventFrame.EventName,
			eventFrame.LocationCountryISO,
			eventFrame.LocationCity,
			eventFrame.ClientTimeStampUTC,
			time.Now().UTC(),
			eventFrame.UserAgent,
			eventFrame.PageURL,
			eventFrame.PagePath,
			eventFrame.Referrer,
			eventFrame.ReferredDomain,
			eventFrame.ReferrerPath,
			eventFrame.UTMParameters,
			eventFrame.ClickOn,
			clickPositionX,
			clickPositionY,
			eventFrame.ProjectID,
			eventFrame.OrganizationID,
		); err != nil {
			log.Printf("Error inserting event: %v", err)
			msg.Nak()
			return
		}

		msg.Ack()
	})

	return err
}

func (p *Processor) Stop() error {
	p.cancelConsumer()
	p.consumerJsConnn.Conn().Close()
	return nil
}

func (p *Processor) processEvent(eventFrame *types.ClientEventFrameV1) error {

	for _, stage := range p.stages {
		if err := stage.ProcessFrame(eventFrame); err != nil {
			return err
		}
	}

	b, err := json.Marshal(eventFrame)
	if err != nil {
		return err
	}

	fmt.Println(string(b))

	return nil
}
