package services

import (
	"fmt"
	"zori/internal/natsstream"

	"github.com/nats-io/nats.go"
)

type EventsService struct {
	natsStream *natsstream.Stream
}

func NewEventsService(natsStream *natsstream.Stream) *EventsService {
	return &EventsService{
		natsStream: natsStream,
	}
}

func (s *EventsService) SubscribeOnEventsStream(projectId string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return s.natsStream.GetConnection().Subscribe(fmt.Sprintf("events:project:%s", projectId), handler)
}
