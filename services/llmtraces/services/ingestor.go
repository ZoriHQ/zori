package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zori/internal/cache"
	"zori/internal/natsstream"
	"zori/services/llmtraces/types"
)

const (
	TraceDedupCacheKey = "llm-trace:dedupe"
	TraceDeduplicationWindow = 5 * time.Second
)

type LLMTraceIngestor struct {
	natsStream   *natsstream.Stream
	cacheService *cache.CacheService
}

func NewLLMTraceIngestor(natsStream *natsstream.Stream, cacheService *cache.CacheService) *LLMTraceIngestor {
	return &LLMTraceIngestor{
		natsStream:   natsStream,
		cacheService: cacheService,
	}
}

func (i *LLMTraceIngestor) Ingest(ctx context.Context, projectID, orgID string, traceRequest *types.LLMTraceRequest) error {
	dedupeKey := fmt.Sprintf("%s:%s:%s:%d",
		TraceDedupCacheKey,
		projectID,
		traceRequest.Model,
		time.Now().UnixNano(),
	)

	isNew, err := i.cacheService.SetNX(ctx, dedupeKey, true, TraceDeduplicationWindow)
	if err != nil {
		fmt.Printf("Warning: trace deduplication check failed: %v\n", err)
	} else if !isNew {
		return fmt.Errorf("duplicate trace detected")
	}

	traceFrame := &types.LLMTraceFrame{
		LLMTraceRequest: traceRequest,
		ProjectID:       projectID,
		OrganizationID:  orgID,
	}

	frameBytes, err := json.Marshal(traceFrame)
	if err != nil {
		return fmt.Errorf("failed to marshal trace frame: %w", err)
	}

	if err := i.natsStream.Publish(natsstream.LLMTracesRawSubject, frameBytes); err != nil {
		return fmt.Errorf("failed to publish trace to NATS: %w", err)
	}

	return nil
}
