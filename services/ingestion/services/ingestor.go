package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"zori/internal/cache"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/storage/postgres/models"
	"zori/services/ingestion/types"
)

const (
	// EventDeduplicationWindow is the time window for deduplicating events based on client_generated_event_id
	EventDeduplicationWindow = 5 * time.Second
)

type Ingestor struct {
	natsStream    *natsstream.Stream
	cacheService  *cache.CacheService
	ingestMetrics *metrics.IngestMetrics
}

func NewIngestor(natsStream *natsstream.Stream, cacheService *cache.CacheService, ingestMetrics *metrics.IngestMetrics) *Ingestor {
	return &Ingestor{
		natsStream:    natsStream,
		cacheService:  cacheService,
		ingestMetrics: ingestMetrics,
	}
}

func (i *Ingestor) Ingest(project *models.Project, clientEvent *types.ClientEventV1) error {
	startTime := time.Now()

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
		i.ingestMetrics.RecordEventDedupe(project.ID, project.OrganizationID, "error")
		// Continue with ingestion even if deduplication fails
	} else if !isNew {
		// Event is a duplicate, skip ingestion
		i.ingestMetrics.RecordEventDedupe(project.ID, project.OrganizationID, "duplicate")
		i.ingestMetrics.RecordIngestRequest(project.ID, project.OrganizationID, "duplicate", time.Since(startTime))
		return fmt.Errorf("duplicate event detected: %s", clientEvent.ClientGeneratedEventID)
	} else {
		i.ingestMetrics.RecordEventDedupe(project.ID, project.OrganizationID, "new")
	}

	eventFrame := types.ClientEventFrameV1{
		ClientEventV1:  clientEvent,
		ProjectID:      project.ID,
		OrganizationID: project.OrganizationID,
	}

	eventFrameBytes, err := json.Marshal(&eventFrame)
	if err != nil {
		i.ingestMetrics.RecordIngestError(project.ID, project.OrganizationID, "marshal_error")
		i.ingestMetrics.RecordIngestRequest(project.ID, project.OrganizationID, "error", time.Since(startTime))
		return err
	}

	if err = i.natsStream.GetConnection().Publish("events:raw", eventFrameBytes); err != nil {
		i.ingestMetrics.RecordIngestError(project.ID, project.OrganizationID, "nats_publish_error")
		i.ingestMetrics.RecordIngestRequest(project.ID, project.OrganizationID, "error", time.Since(startTime))
		return err
	}

	// Determine event type based on event name
	eventType := "track"
	eventName := ""
	if clientEvent.EventName != nil {
		eventName = *clientEvent.EventName
		if eventName == "identify" {
			eventType = "identify"
		}
	}

	i.ingestMetrics.RecordEvent(project.ID, project.OrganizationID, eventName, eventType)
	i.ingestMetrics.RecordIngestRequest(project.ID, project.OrganizationID, "success", time.Since(startTime))

	return nil
}
