package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"zori/internal/cache"
	"zori/internal/natsstream"
	"zori/internal/storage/postgres/models"
	"zori/services/ingestion/types"
)

const (
	// EventDeduplicationWindow is the time window for deduplicating events based on client_generated_event_id
	EventDeduplicationWindow = 5 * time.Second
)

type Ingestor struct {
	natsStream   *natsstream.Stream
	cacheService *cache.CacheService
}

func NewIngestor(natsStream *natsstream.Stream, cacheService *cache.CacheService) *Ingestor {
	return &Ingestor{
		natsStream:   natsStream,
		cacheService: cacheService,
	}
}

func (i *Ingestor) Ingest(project *models.Project, clientEvent *types.ClientEventV1) error {
	// Deduplicate events based on client-generated event ID
	// Use Redis SetNX to atomically check and set the deduplication key
	dedupeKey := cache.EventDedupeCacheKey.FromValue(
		fmt.Sprintf("%s:%s", project.ID, clientEvent.ClientGeneratedEventID),
	)

	// Try to set the key with the deduplication window as TTL
	// If the key already exists, this event is a duplicate
	isNew, err := i.cacheService.SetNX(
		context.Background(),
		dedupeKey,
		true, // Just a marker value
		EventDeduplicationWindow,
	)
	if err != nil {
		// Log error but don't fail the request - deduplication is best-effort
		fmt.Printf("Warning: event deduplication check failed: %v\n", err)
		// Continue with ingestion even if deduplication fails
	} else if !isNew {
		// Event is a duplicate, skip ingestion
		return fmt.Errorf("duplicate event detected: %s", clientEvent.ClientGeneratedEventID)
	}

	eventFrame := types.ClientEventFrameV1{
		ClientEventV1:  clientEvent,
		ProjectID:      project.ID,
		OrganizationID: project.OrganizationID,
	}

	eventFrameBytes, err := json.Marshal(&eventFrame)
	if err != nil {
		return err
	}

	if err = i.natsStream.GetConnection().Publish("events:raw", eventFrameBytes); err != nil {
		return err
	}

	return nil
}
