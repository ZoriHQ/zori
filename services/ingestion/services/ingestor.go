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

	dedupeKey := cache.EventDedupeCacheKey.FromValue(
		fmt.Sprintf("%s:%s", project.ID, clientEvent.ClientGeneratedEventID),
	)

	isNew, err := i.cacheService.SetNX(
		context.Background(),
		dedupeKey,
		true,
		EventDeduplicationWindow,
	)
	if err != nil {
		fmt.Printf("Warning: event deduplication check failed: %v\n", err)
		i.ingestMetrics.RecordEventDedupe(project.ID, project.OrganizationID, "error")
	} else if !isNew {
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
