package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
	"zori/internal/cache"
	"zori/internal/config"
	"zori/internal/natsstream"
	"zori/internal/telemetry"
	"zori/services/live/types"

	"github.com/nats-io/nats.go"
)

const (
	DefaultSessionTTL        = 30 * time.Second
	DefaultCleanupInterval   = 30 * time.Second
	liveVisitorSubjectPrefix = "live:visitors:"
)

type LiveSessionTracker struct {
	cacheService    *cache.CacheService
	natsStream      *natsstream.Stream
	logger          *telemetry.Logger
	sessionTTL      time.Duration
	cleanupInterval time.Duration

	previousCounts map[string]int64
	countsMu       sync.RWMutex

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewLiveSessionTracker(
	cacheService *cache.CacheService,
	natsStream *natsstream.Stream,
	logger *telemetry.Logger,
	cfg *config.Config,
) *LiveSessionTracker {
	sessionTTL := DefaultSessionTTL
	if cfg.LiveSessionTTLSeconds > 0 {
		sessionTTL = time.Duration(cfg.LiveSessionTTLSeconds) * time.Second
	}

	cleanupInterval := DefaultCleanupInterval
	if cfg.LiveCleanupIntervalSeconds > 0 {
		cleanupInterval = time.Duration(cfg.LiveCleanupIntervalSeconds) * time.Second
	}

	return &LiveSessionTracker{
		cacheService:    cacheService,
		natsStream:      natsStream,
		logger:          logger,
		sessionTTL:      sessionTTL,
		cleanupInterval: cleanupInterval,
		previousCounts:  make(map[string]int64),
		stopCh:          make(chan struct{}),
	}
}

func (t *LiveSessionTracker) Start() error {
	t.wg.Add(1)
	go t.cleanupLoop()
	t.logger.Info("LiveSessionTracker started",
		telemetry.String("session_ttl", t.sessionTTL.String()),
		telemetry.String("cleanup_interval", t.cleanupInterval.String()),
	)
	return nil
}

func (t *LiveSessionTracker) Stop() error {
	close(t.stopCh)
	t.wg.Wait()
	t.logger.Info("LiveSessionTracker stopped")
	return nil
}

func (t *LiveSessionTracker) TrackSession(ctx context.Context, projectID, sessionID string) error {
	key := t.sessionKey(projectID)
	score := float64(time.Now().Unix())

	wasNew, err := t.cacheService.ZAdd(ctx, key, score, sessionID)
	if err != nil {
		t.logger.Error("Failed to track session",
			telemetry.Error(err),
			telemetry.String("project_id", projectID),
			telemetry.String("session_id", sessionID),
		)
		return err
	}

	if wasNew {
		if err := t.checkAndPublishUpdate(ctx, projectID); err != nil {
			t.logger.Error("Failed to publish live update",
				telemetry.Error(err),
				telemetry.String("project_id", projectID),
			)
		}
	}

	return nil
}

func (t *LiveSessionTracker) RemoveSession(ctx context.Context, projectID, sessionID string) error {
	key := t.sessionKey(projectID)

	wasRemoved, err := t.cacheService.ZRem(ctx, key, sessionID)
	if err != nil {
		t.logger.Error("Failed to remove session",
			telemetry.Error(err),
			telemetry.String("project_id", projectID),
			telemetry.String("session_id", sessionID),
		)
		return err
	}

	if wasRemoved {
		if err := t.checkAndPublishUpdate(ctx, projectID); err != nil {
			t.logger.Error("Failed to publish live update after removal",
				telemetry.Error(err),
				telemetry.String("project_id", projectID),
			)
		}
	}

	return nil
}

func (t *LiveSessionTracker) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	key := t.sessionKey(projectID)
	return t.cacheService.ZCard(ctx, key)
}

func (t *LiveSessionTracker) SubscribeToUpdates(projectID string, handler func(count *types.LiveVisitorCount)) (func() error, error) {
	subject := t.natsSubject(projectID)

	sub, err := t.natsStream.GetConnection().Subscribe(subject, func(msg *nats.Msg) {
		var count types.LiveVisitorCount
		if err := json.Unmarshal(msg.Data, &count); err != nil {
			t.logger.Error("Failed to unmarshal live visitor count",
				telemetry.Error(err),
				telemetry.String("project_id", projectID),
			)
			return
		}
		handler(&count)
	})
	if err != nil {
		return nil, err
	}

	return sub.Unsubscribe, nil
}

func (t *LiveSessionTracker) cleanupLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.cleanupStaleSessions()
		}
	}
}

func (t *LiveSessionTracker) cleanupStaleSessions() {
	ctx := context.Background()

	lockTTL := t.cleanupInterval * 2 / 3
	if lockTTL < 1*time.Second {
		lockTTL = 1 * time.Second
	}

	lockKey := string(cache.LiveCleanupLockCacheKey)
	acquired, err := t.cacheService.SetNX(ctx, lockKey, "locked", lockTTL)
	if err != nil {
		t.logger.Error("Failed to acquire cleanup lock", telemetry.Error(err))
		return
	}
	if !acquired {
		t.logger.Debug("Skipping cleanup - another instance holds the lock")
		return
	}

	projectIDs, err := t.discoverLiveSessionKeys(ctx)
	if err != nil {
		t.logger.Error("Failed to discover live session keys", telemetry.Error(err))
		return
	}

	cutoffTime := time.Now().Add(-t.sessionTTL).Unix()
	maxScore := strconv.FormatInt(cutoffTime, 10)

	for _, projectID := range projectIDs {
		key := t.sessionKey(projectID)

		removed, err := t.cacheService.ZRemRangeByScore(ctx, key, "-inf", maxScore)
		if err != nil {
			t.logger.Error("Failed to cleanup stale sessions",
				telemetry.Error(err),
				telemetry.String("project_id", projectID),
			)
			continue
		}

		if removed > 0 {
			t.logger.Debug("Cleaned up stale sessions",
				telemetry.String("project_id", projectID),
				telemetry.Int("removed_count", int(removed)),
			)
			if err := t.checkAndPublishUpdate(ctx, projectID); err != nil {
				t.logger.Error("Failed to publish update after cleanup",
					telemetry.Error(err),
					telemetry.String("project_id", projectID),
				)
			}
		}
	}
}

func (t *LiveSessionTracker) checkAndPublishUpdate(ctx context.Context, projectID string) error {
	currentCount, err := t.GetActiveCount(ctx, projectID)
	if err != nil {
		return err
	}

	t.countsMu.Lock()
	previousCount, exists := t.previousCounts[projectID]
	if !exists || previousCount != currentCount {
		t.previousCounts[projectID] = currentCount
		t.countsMu.Unlock()

		return t.publishUpdate(projectID, currentCount)
	}
	t.countsMu.Unlock()

	return nil
}

func (t *LiveSessionTracker) publishUpdate(projectID string, count int64) error {
	update := types.LiveVisitorCount{
		ProjectID: projectID,
		Count:     count,
	}

	data, err := json.Marshal(update)
	if err != nil {
		return err
	}

	subject := t.natsSubject(projectID)
	if err := t.natsStream.GetConnection().Publish(subject, data); err != nil {
		return err
	}

	t.logger.Debug("Published live visitor count update",
		telemetry.String("project_id", projectID),
		telemetry.Int("count", int(count)),
	)

	return nil
}

func (t *LiveSessionTracker) sessionKey(projectID string) string {
	return cache.LiveSessionsCacheKey.FromValue(projectID)
}

func (t *LiveSessionTracker) natsSubject(projectID string) string {
	return fmt.Sprintf("%s%s", liveVisitorSubjectPrefix, projectID)
}

func (t *LiveSessionTracker) discoverLiveSessionKeys(ctx context.Context) ([]string, error) {
	pattern := string(cache.LiveSessionsCacheKey) + ":*"
	keys, err := t.cacheService.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	prefix := string(cache.LiveSessionsCacheKey) + ":"
	projectIDs := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key) > len(prefix) {
			projectID := key[len(prefix):]
			projectIDs = append(projectIDs, projectID)
		}
	}

	return projectIDs, nil
}
