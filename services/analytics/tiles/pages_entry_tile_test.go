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

func TestEntryPagesTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewEntryPagesTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should aggregate sessions by entry page", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
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
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-1*time.Hour + 5*time.Minute),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/about",
				PagePath:           "/about",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/pricing",
				PagePath:           "/pricing",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-2*time.Hour + 3*time.Minute),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/contact",
				PagePath:           "/contact",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-3",
				SessionID:          "session-3",
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
				VisitorID:          "visitor-old",
				SessionID:          "session-old",
				ClientTimestampUTC: now.Add(-10 * 24 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/blog",
				PagePath:           "/blog",
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
		require.NotEmpty(t, response.Data)

		assert.GreaterOrEqual(t, len(response.Data), 2, "Should have at least 2 entry pages")

		var homePageCount uint64
		var pricingPageCount uint64
		for _, data := range response.Data {
			if data.Page == "/" {
				homePageCount = data.Count
			} else if data.Page == "/pricing" {
				pricingPageCount = data.Count
			}
		}

		assert.Equal(t, uint64(2), homePageCount, "Home page should have 2 sessions")
		assert.Equal(t, uint64(1), pricingPageCount, "Pricing page should have 1 session")

		t.Logf("Entry pages data: %+v", response.Data)
	})

	t.Run("should return empty when no sessions exist", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		tile2 := tiles.NewEntryPagesTile(tc2.ClickHouse)

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
		assert.Empty(t, response.Data, "Should have no entry pages when no sessions")
	})
}
