package natsstream

import (
	"errors"
	"zori/internal/config"

	"github.com/nats-io/nats.go"
)

type Stream struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewStream(conf *config.Config) *Stream {
	var (
		nc  *nats.Conn
		err error
	)
	if conf.NatsCredentialsContent != "" {
		nc, err = nats.Connect(conf.NatsStreamURL, nats.UserCredentialBytes([]byte(conf.NatsCredentialsContent)))
		if err != nil {
			panic(err)
		}
	} else {
		nc, err = nats.Connect(conf.NatsStreamURL)
		if err != nil {
			panic(err)
		}
	}

	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}

	return &Stream{
		nc: nc,
		js: js,
	}
}

func (s *Stream) UpsertJetStream(name string, sourceSubject string) error {
	streamInfo, err := s.js.StreamInfo(name)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}
	if streamInfo == nil {
		streamInfo, err = s.js.AddStream(&nats.StreamConfig{
			Name:     name,
			Subjects: []string{sourceSubject},
			MaxBytes: 100000,
		})
		if err != nil {
			return err
		}
	}

	return err
}

func (s *Stream) GetJetStream() nats.JetStreamContext {
	return s.js
}

func (s *Stream) GetConnection() *nats.Conn {
	return s.nc
}
