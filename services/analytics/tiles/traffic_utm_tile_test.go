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

// NOTE: TrafficSourceTile currently only implements FetchByReferer and FetchByCountry
// The UTMCampaignSourceData struct is defined but there's no FetchByUTM method yet.
// This test file tests the existing functionality and can be extended when UTM-specific
// methods are added.

func TestTrafficSourceTile_FetchByReferer(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewTrafficSourceTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should aggregate traffic by referrer (TrafficSourceTile)", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-twitter-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				ReferrerDomain:     nullable.FromString("twitter.com").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-twitter-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				ReferrerDomain:     nullable.FromString("twitter.com").Ptr(),
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

		response, err := tile.FetchByReferer(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.NotEmpty(t, response.Data)

		var twitterData *tiles.RefererTrafficSourceData
		for i := range response.Data {
			if response.Data[i].RefererURL == "twitter.com" {
				twitterData = &response.Data[i]
				break
			}
		}

		require.NotNil(t, twitterData)
		assert.NotNil(t, twitterData.Count)
		assert.Equal(t, uint64(2), *twitterData.Count)
	})
}

func TestTrafficSourceTile_FetchByCountry(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewTrafficSourceTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should aggregate traffic by country (TrafficSourceTile)", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-jp-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				LocationCountryISO: nullable.FromString("JP").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-jp-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				LocationCountryISO: nullable.FromString("JP").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-jp-3",
				SessionID:          "session-3",
				ClientTimestampUTC: now.Add(-3 * time.Hour),
				LocationCountryISO: nullable.FromString("JP").Ptr(),
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

		response, err := tile.FetchByCountry(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.NotEmpty(t, response.Data)

		var japanData *tiles.CountryTrafficSourceData
		for i := range response.Data {
			if response.Data[i].Country == "JP" {
				japanData = response.Data[i]
				break
			}
		}

		require.NotNil(t, japanData)
		assert.Equal(t, uint64(3), japanData.Count)
	})
}

// TODO: Add tests for UTM-specific functionality when FetchByUTM method is implemented
// The UTMCampaignSourceData struct is defined but not currently used.
// Expected test cases:
// - Test UTM source aggregation
// - Test UTM medium aggregation
// - Test UTM campaign aggregation
// - Test revenue attribution by UTM parameters
// - Test combinations of UTM parameters
