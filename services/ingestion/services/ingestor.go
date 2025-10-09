package services

import (
	"zori/internal/natsstream"
)

type Ingestor struct {
	natsStream *natsstream.Stream
}

func NewIngestor(natsStream *natsstream.Stream) *Ingestor {
	return &Ingestor{
		natsStream: natsStream,
	}
}

func (i *Ingestor) Ingest(data []byte) error {
	if _, err := i.natsStream.GetJetStream().Publish("events:raw", data); err != nil {
		return err
	}

	return nil
}
