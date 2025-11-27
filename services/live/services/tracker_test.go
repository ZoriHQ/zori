package services

import (
	"context"
	"testing"
	"time"
	"zori/internal/cache"
	"zori/internal/config"
	"zori/internal/natsstream"
	"zori/internal/telemetry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestTracker(t *testing.T) (*LiveSessionTracker, func()) {
	cfg := &config.Config{
		RedisADDS:                  "localhost:6379",
		RedisPASS:                  "",
		NatsStreamURL:              "nats://localhost:4222",
		LiveSessionTTLSeconds:      2, // Short TTL for testing
		LiveCleanupIntervalSeconds: 1, // Fast cleanup for testing
	}

	cacheService := cache.NewCacheService(cfg)
	natsStream := natsstream.NewStream(cfg)

	logger, err := telemetry.NewLogger()
	require.NoError(t, err)

	tracker := NewLiveSessionTracker(cacheService, natsStream, logger, cfg)

	cleanup := func() {
		_ = tracker.Stop()
		_ = cacheService.FlushAll(context.Background())
	}

	return tracker, cleanup
}

func TestLiveSessionTrackerCleanup(t *testing.T) {
	tracker, cleanup := setupTestTracker(t)
	defer cleanup()

	ctx := context.Background()

	err := tracker.Start()
	require.NoError(t, err)

	projectID1 := "test-project-1"
	projectID2 := "test-project-2"
	sessionID1 := "session-1"
	sessionID2 := "session-2"
	sessionID3 := "session-3"

	err = tracker.TrackSession(ctx, projectID1, sessionID1)
	require.NoError(t, err)

	err = tracker.TrackSession(ctx, projectID1, sessionID2)
	require.NoError(t, err)

	err = tracker.TrackSession(ctx, projectID2, sessionID3)
	require.NoError(t, err)

	count1, err := tracker.GetActiveCount(ctx, projectID1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count1)

	count2, err := tracker.GetActiveCount(ctx, projectID2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count2)

	time.Sleep(6 * time.Second)

	count1After, err := tracker.GetActiveCount(ctx, projectID1)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count1After, "Sessions should have been cleaned up for project 1")

	count2After, err := tracker.GetActiveCount(ctx, projectID2)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count2After, "Sessions should have been cleaned up for project 2")
}

func TestLiveSessionTrackerRefreshesTimestamp(t *testing.T) {
	tracker, cleanup := setupTestTracker(t)
	defer cleanup()

	ctx := context.Background()

	// Start the tracker
	err := tracker.Start()
	require.NoError(t, err)

	projectID := "test-project"
	sessionID := "session-1"

	// Track session at T=0
	err = tracker.TrackSession(ctx, projectID, sessionID)
	require.NoError(t, err)

	// Wait 1.5s (less than TTL of 2s)
	time.Sleep(1500 * time.Millisecond)

	// Refresh the session - this should update the timestamp to T=1.5
	err = tracker.TrackSession(ctx, projectID, sessionID)
	require.NoError(t, err)

	// Wait 1.5s more (now at T=3)
	// Session was refreshed at T=1.5, so cutoff at T=3-2=T=1 means session with score=1.5 should survive
	time.Sleep(1500 * time.Millisecond)

	count, err := tracker.GetActiveCount(ctx, projectID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "Session should still be active after timestamp refresh")

	// Wait another 2s for the session to expire (now at T=5, cutoff=T=3, score=1.5 < 3)
	time.Sleep(2 * time.Second)

	countAfter, err := tracker.GetActiveCount(ctx, projectID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), countAfter, "Session should be cleaned up after full TTL")
}

func TestDiscoverLiveSessionKeys(t *testing.T) {
	tracker, cleanup := setupTestTracker(t)
	defer cleanup()

	ctx := context.Background()

	projectID1 := "project-1"
	projectID2 := "project-2"
	projectID3 := "project-3"

	err := tracker.TrackSession(ctx, projectID1, "session-1")
	require.NoError(t, err)

	err = tracker.TrackSession(ctx, projectID2, "session-2")
	require.NoError(t, err)

	err = tracker.TrackSession(ctx, projectID3, "session-3")
	require.NoError(t, err)

	projectIDs, err := tracker.discoverLiveSessionKeys(ctx)
	require.NoError(t, err)

	assert.Len(t, projectIDs, 3)
	assert.Contains(t, projectIDs, projectID1)
	assert.Contains(t, projectIDs, projectID2)
	assert.Contains(t, projectIDs, projectID3)
}
