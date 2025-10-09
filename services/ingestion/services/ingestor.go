package services

import (
	"encoding/json"
	"zori/internal/natsstream"
	"zori/internal/storage/postgres/models"
	"zori/services/ingestion/types"
)

type Ingestor struct {
	natsStream *natsstream.Stream
}

func NewIngestor(natsStream *natsstream.Stream) *Ingestor {
	return &Ingestor{
		natsStream: natsStream,
	}
}

func (i *Ingestor) Ingest(project *models.Project, clientEvent *types.ClientEventV1) error {
	eventFrame := types.ClientEventFrameV1{
		ClientEventV1:  clientEvent,
		ProjectID:      project.ID,
		OrganizationID: project.OrganizationID,
	}

	eventFrameBytes, err := json.Marshal(&eventFrame)
	if err != nil {
		return err
	}

	if _, err := i.natsStream.GetJetStream().Publish("events:raw", eventFrameBytes); err != nil {
		return err
	}

	return nil
}
