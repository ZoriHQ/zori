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
	RecordingDeduplicationWindow = 5 * time.Second
)

type Recorder struct {
	natsStream   *natsstream.Stream
	cacheService *cache.CacheService
}

func NewRecorder(natsStream *natsstream.Stream, cacheService *cache.CacheService) *Recorder {
	return &Recorder{
		natsStream:   natsStream,
		cacheService: cacheService,
	}
}

func (r *Recorder) Record(project *models.Project, recordingEvent *types.RecordingEventV1) error {
	dedupeKey := cache.EventDedupeCacheKey.FromValue(
		fmt.Sprintf("rec:%s:%s", project.ID, recordingEvent.ClientGeneratedEventID),
	)

	isNew, err := r.cacheService.SetNX(
		context.Background(),
		dedupeKey,
		true,
		RecordingDeduplicationWindow,
	)
	if err != nil {
		fmt.Printf("Warning: recording deduplication check failed: %v\n", err)
	} else if !isNew {
		return fmt.Errorf("duplicate recording event detected: %s", recordingEvent.ClientGeneratedEventID)
	}

	eventFrame := types.RecordingEventFrameV1{
		RecordingEventV1: recordingEvent,
		ProjectID:        project.ID,
		OrganizationID:   project.OrganizationID,
		ChunkIndex:       0,
		IsFinalChunk:     false,
	}

	eventFrameBytes, err := json.Marshal(&eventFrame)
	if err != nil {
		return fmt.Errorf("failed to marshal recording event: %w", err)
	}

	if err = r.natsStream.GetConnection().Publish("recordings:raw", eventFrameBytes); err != nil {
		return fmt.Errorf("failed to publish recording event: %w", err)
	}

	return nil
}
