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

func TestVisitorsByOSTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewVisitorsByOSTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should aggregate visitors by OS", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-google-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				OSName:             nullable.FromString("MacOS").Ptr(),
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-google-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				OSName:             nullable.FromString("Windows").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-facebook-1",
				SessionID:          "session-3",
				ClientTimestampUTC: now.Add(-3 * time.Hour),
				OSName:             nullable.FromString("Windows").Ptr(),
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
			TimeBoundaries: filters.TimeBoundariesToday,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project.OrganizationID)

		response, err := tile.Fetch(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.NotEmpty(t, response.Data)

		var windowsVisitorCount uint64
		var macosVisitorCount uint64
		for _, data := range response.Data {
			switch data.BrowserName {
			case "Windows":
				windowsVisitorCount += data.Count
			case "MacOS":
				macosVisitorCount += data.Count
			}
		}

		require.NotNil(t, windowsVisitorCount, "Windows visitor count should be present")
		assert.NotNil(t, windowsVisitorCount)
		assert.Equal(t, uint64(2), windowsVisitorCount)

		require.NotNil(t, macosVisitorCount, "MacOS visitor count should be present")
		assert.NotNil(t, macosVisitorCount)
		assert.Equal(t, uint64(1), macosVisitorCount)
	})
}
