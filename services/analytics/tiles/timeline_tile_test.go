package tiles_test

import (
	"testing"
	"time"

	"zori/di"
	"zori/services/analytics/filters"
	"zori/services/analytics/tiles"
	"zori/testutil/fixtures"

	"github.com/Cleverse/go-utilities/nullable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimelineTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewTimelineTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should aggregate timeline data with device types", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-6 * 24 * time.Hour),
				DeviceType:         nullable.FromString("Desktop").Ptr(),
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
				ClientTimestampUTC: now.Add(-6 * 24 * time.Hour),
				DeviceType:         nullable.FromString("Desktop").Ptr(),
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
				ClientTimestampUTC: now.Add(-5 * 24 * time.Hour),
				DeviceType:         nullable.FromString("Mobile").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-4",
				SessionID:          "session-4",
				ClientTimestampUTC: now.Add(-5 * 24 * time.Hour),
				DeviceType:         nullable.FromString("Mobile").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-5",
				SessionID:          "session-5",
				ClientTimestampUTC: now.Add(-5 * 24 * time.Hour),
				DeviceType:         nullable.FromString("Mobile").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-6",
				SessionID:          "session-6",
				ClientTimestampUTC: now.Add(-4 * 24 * time.Hour),
				DeviceType:         nil,
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

		assert.Equal(t, tiles.CardPrecisionDaily, response.Precision)

		assert.NotEmpty(t, response.Data, "Should have timeline data points")

		assert.GreaterOrEqual(t, response.TotalVisits, uint64(6), "Should have at least 6 total visits")

		var foundDesktop, foundMobile, foundUnknown bool
		for _, dp := range response.Data {
			if dp.NumDesktopVisits == 2 {
				foundDesktop = true
			}
			if dp.NumMobileVisits == 3 {
				foundMobile = true
			}
			if dp.NumUnknownVisits == 1 {
				foundUnknown = true
			}
		}

		assert.True(t, foundDesktop, "Should find day with 2 desktop visits")
		assert.True(t, foundMobile, "Should find day with 3 mobile visits")
		assert.True(t, foundUnknown, "Should find day with 1 unknown visit")

		t.Logf("Timeline data points: %d", len(response.Data))
		t.Logf("Total visits: %d", response.TotalVisits)
	})

	t.Run("should handle empty data gracefully", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		tile3 := tiles.NewTimelineTile(tc3.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project3.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project3.OrganizationID)

		response, err := tile3.Fetch(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.NotEmpty(t, response.Data, "Should have timeline data points even with no events")
		assert.Equal(t, uint64(0), response.TotalVisits, "Total visits should be 0")

		for _, dp := range response.Data {
			assert.Equal(t, uint64(0), dp.NumDesktopVisits)
			assert.Equal(t, uint64(0), dp.NumMobileVisits)
			assert.Equal(t, uint64(0), dp.NumUnknownVisits)
		}
	})

	t.Run("should use correct precision based on time boundaries", func(t *testing.T) {
		testCases := []struct {
			timeBoundary      filters.TimeBoundaries
			expectedPrecision tiles.CardPrecision
		}{
			{filters.TimeBoundariesHour, tiles.CardPrecisionMinutes},
			{filters.TimeBoundariesToday, tiles.CardPrecisionHourly},
			{filters.TimeBoundariesLastWeek, tiles.CardPrecisionDaily},
			{filters.TimeBoundariesLastMonth, tiles.CardPrecisionWeekly},
			{filters.TimeBoundariesLast90Days, tiles.CardPrecisionMonthly},
		}

		for _, testCase := range testCases {
			t.Run(string(testCase.timeBoundary), func(t *testing.T) {
				tc4 := di.NewTestContainer(t)
				defer tc4.Cleanup()

				_, project4 := fixtures.SetupAccountAndProject(t, tc4)
				tile4 := tiles.NewTimelineTile(tc4.ClickHouse)

				time.Sleep(500 * time.Millisecond)

				timeRange, err := filters.ValidateTimeRange(testCase.timeBoundary)
				require.NoError(t, err)

				filter := &filters.SectionFilter{
					ProjectID:      project4.ID,
					TimeBoundaries: testCase.timeBoundary,
					TimeRange:      timeRange,
				}

				appCtx := fixtures.NewTestCtxWithOrg(project4.OrganizationID)

				response, err := tile4.Fetch(appCtx, filter)
				require.NoError(t, err)
				require.NotNil(t, response)

				assert.Equal(t, testCase.expectedPrecision, response.Precision,
					"Precision should match time boundary: %s", testCase.timeBoundary)
			})
		}
	})
}
