package services

import (
	"context"
	"encoding/json"
	"zori/internal/natsstream"
	"zori/internal/storage/postgres/models"
	"zori/services/ingestion/data"
	"zori/services/ingestion/types"
)

type Identifier struct {
	natsStream        *natsstream.Stream
	visitorRepository *data.VisitorRepository
}

func NewIdentifier(natsStream *natsstream.Stream, visitorRepository *data.VisitorRepository) *Identifier {
	return &Identifier{
		natsStream:        natsStream,
		visitorRepository: visitorRepository,
	}
}

// Identify processes an identify event and stores visitor identity in PostgreSQL
func (i *Identifier) Identify(ctx context.Context, project *models.Project, identifyEvent *types.IdentifyEventV1) error {
	// Create visitor model from identify event
	visitor := &models.Visitor{
		VisitorID:      identifyEvent.VisitorID,
		ProjectID:      project.ID,
		OrganizationID: project.OrganizationID,
		UserID:         identifyEvent.AppID,
		Email:          identifyEvent.Email,
		Name:           identifyEvent.Fullname,
		Phone:          identifyEvent.Phone,
	}

	if identifyEvent.AdditionalProperties != nil && len(identifyEvent.AdditionalProperties) > 0 {
		visitor.CustomTraits = models.JSONB(identifyEvent.AdditionalProperties)
	}

	if err := i.visitorRepository.UpsertVisitor(ctx, visitor); err != nil {
		return err
	}

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
