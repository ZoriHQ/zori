package tiles_test

import (
	"testing"
	"time"

	"zori/di"
	"zori/services/analytics/filters"
	"zori/services/analytics/tiles"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBounceRateTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewBounceRateTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should calculate bounce rate correctly", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-1",
				SessionID:          "session-bounce-1",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-2",
				SessionID:          "session-multi-1",
				ClientTimestampUTC: now.Add(-3 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-2",
				SessionID:          "session-multi-1",
				ClientTimestampUTC: now.Add(-3*time.Hour - 5*time.Minute),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/page2",
				PagePath:           "/page2",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-3",
				SessionID:          "session-bounce-2",
				ClientTimestampUTC: now.Add(-4 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/about",
				PagePath:           "/about",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-old-1",
				SessionID:          "session-old-bounce-1",
				ClientTimestampUTC: now.Add(-10 * 24 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
		}

		err := fixtures.InsertEventsDirect(t, tc, events)
		require.NoError(t, err)

		time.Sleep(1500 * time.Millisecond)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project.OrganizationID)

		response, err := tile.Fetch(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.InDelta(t, 66.66, response.Rate, 0.1, "Should have ~66.66% bounce rate (2 out of 3 sessions)")
		assert.InDelta(t, 100.0, response.PreviousRate, 0.1, "Should have 100% bounce rate in previous period (1 out of 1 sessions)")

		t.Logf("Current bounce rate: %.2f%%, Previous: %.2f%%", response.Rate, response.PreviousRate)
	})

	t.Run("should return zero when no sessions exist", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		tile2 := tiles.NewBounceRateTile(tc2.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project2.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project2.OrganizationID)

		response, err := tile2.Fetch(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Equal(t, 0.0, response.Rate, "Should have 0% bounce rate when no sessions")
		assert.Equal(t, 0.0, response.PreviousRate, "Should have 0% previous bounce rate when no sessions")
	})
}
