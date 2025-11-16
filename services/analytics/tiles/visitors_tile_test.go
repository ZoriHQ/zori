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

func TestUniqueVisitorsTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewUniqueVisitorsTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should count unique visitors in current and previous periods", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-1",
				SessionID:          "session-1",
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
				SessionID:          "session-2",
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
				VisitorID:          "visitor-3",
				SessionID:          "session-3",
				ClientTimestampUTC: now.Add(-4 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-1",
				SessionID:          "session-1-2",
				ClientTimestampUTC: now.Add(-5 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/page2",
				PagePath:           "/page2",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-old-1",
				SessionID:          "session-old-1",
				ClientTimestampUTC: now.Add(-10 * 24 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-old-2",
				SessionID:          "session-old-2",
				ClientTimestampUTC: now.Add(-11 * 24 * time.Hour),
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

		time.Sleep(1 * time.Second)

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

		assert.Equal(t, uint64(3), response.Count, "Should have 3 unique visitors in current period")
		assert.Equal(t, uint64(2), response.PreviousCount, "Should have 2 unique visitors in previous period")

		t.Logf("Current unique visitors: %d, Previous: %d", response.Count, response.PreviousCount)
	})

	t.Run("should return zero when no events exist", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		tile2 := tiles.NewUniqueVisitorsTile(tc2.ClickHouse)

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

		assert.Equal(t, uint64(0), response.Count, "Should have 0 unique visitors when no events")
		assert.Equal(t, uint64(0), response.PreviousCount, "Should have 0 previous unique visitors when no events")
	})
}
