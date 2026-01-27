package services

import (
	"context"
	"encoding/json"
	"zori/internal/natsstream"
	"zori/internal/storage/postgres/models"
	"zori/services/ingestion/types"
)

type Identifier struct {
	natsStream *natsstream.Stream
}

func NewIdentifier(natsStream *natsstream.Stream) *Identifier {
	return &Identifier{
		natsStream: natsStream,
	}
}

func (i *Identifier) Identify(ctx context.Context, project *models.Project, identifyEvent *types.IdentifyEventV1) error {
	eventFrame := types.IdentifyEventFrameV1{
		IdentifyEventV1: identifyEvent,
		ProjectID:       project.ID,
		OrganizationID:  project.OrganizationID,
	}

	eventFrameBytes, err := json.Marshal(&eventFrame)
	if err != nil {
		return err
	}

	if err = i.natsStream.GetConnection().Publish("events:identify", eventFrameBytes); err != nil {
		return err
	}

	return nil
}
