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

func TestVisitorsByBrowserTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewVisitorsByBrowserTile(tc.ClickHouse)

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
				BrowserName:        nullable.FromString("Safari").Ptr(),
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
				BrowserName:        nullable.FromString("Edge").Ptr(),
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
				BrowserName:        nullable.FromString("Edge").Ptr(),
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

		var edgeBrowserCount uint64
		var safariBrowserCount uint64
		for _, data := range response.Data {
			if data.BrowserName == "Edge" {
				edgeBrowserCount += data.Count
			} else if data.BrowserName == "Safari" {
				safariBrowserCount += data.Count
			}
		}

		require.NotNil(t, edgeBrowserCount, "Edge browser count should be present")
		assert.NotNil(t, edgeBrowserCount)
		assert.Equal(t, uint64(2), edgeBrowserCount)

		require.NotNil(t, safariBrowserCount, "Safari browser count should be present")
		assert.NotNil(t, safariBrowserCount)
		assert.Equal(t, uint64(1), safariBrowserCount)
	})
}
