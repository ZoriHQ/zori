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

func TestTrafficRefererSourceTile_FetchByReferer(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewTrafficRefererSourceTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should aggregate traffic by referrer domain", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-google-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				ReferrerDomain:     nullable.FromString("google.com").Ptr(),
				ReferrerURL:        nullable.FromString("https://google.com/search?q=test").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-google-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-3 * time.Hour),
				ReferrerDomain:     nullable.FromString("google.com").Ptr(),
				ReferrerURL:        nullable.FromString("https://google.com/search?q=example").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-google-3",
				SessionID:          "session-3",
				ClientTimestampUTC: now.Add(-4 * time.Hour),
				ReferrerDomain:     nullable.FromString("google.com").Ptr(),
				ReferrerURL:        nullable.FromString("https://google.com/search").Ptr(),
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
				SessionID:          "session-4",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				ReferrerDomain:     nullable.FromString("facebook.com").Ptr(),
				ReferrerURL:        nullable.FromString("https://facebook.com").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-facebook-2",
				SessionID:          "session-5",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				ReferrerDomain:     nullable.FromString("facebook.com").Ptr(),
				ReferrerURL:        nullable.FromString("https://facebook.com").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-google-old",
				SessionID:          "session-6",
				ClientTimestampUTC: now.Add(-10 * 24 * time.Hour),
				ReferrerDomain:     nullable.FromString("google.com").Ptr(),
				ReferrerURL:        nullable.FromString("https://google.com").Ptr(),
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

		assert.NotEmpty(t, response.Data, "Should have referrer traffic data")

		var googleData, facebookData *tiles.RefererTrafficSourceData
		for i := range response.Data {
			if response.Data[i].RefererURL == "google.com" {
				googleData = &response.Data[i]
			}
			if response.Data[i].RefererURL == "facebook.com" {
				facebookData = &response.Data[i]
			}
		}

		require.NotNil(t, googleData, "Should have Google referrer data")
		assert.NotNil(t, googleData.Count, "Google current count should not be nil")
		assert.Equal(t, uint64(3), *googleData.Count, "Google should have 3 current visitors")
		assert.NotNil(t, googleData.PreviousCount, "Google previous count should not be nil")
		assert.Equal(t, uint64(1), *googleData.PreviousCount, "Google should have 1 previous visitor")

		require.NotNil(t, facebookData, "Should have Facebook referrer data")
		assert.NotNil(t, facebookData.Count, "Facebook current count should not be nil")
		assert.Equal(t, uint64(2), *facebookData.Count, "Facebook should have 2 current visitors")

		t.Logf("Google: current=%d, previous=%d", *googleData.Count, *googleData.PreviousCount)
		t.Logf("Facebook: current=%d", *facebookData.Count)
	})

	t.Run("should handle DIRECT/NONE for missing referrers", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		tile2 := tiles.NewTrafficRefererSourceTile(tc2.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-direct-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				ReferrerDomain:     nil,
				ReferrerURL:        nil,
				ProjectID:          project2.ID,
				OrganizationID:     project2.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project2.Domain,
			},
			{
				VisitorID:          "visitor-direct-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				ReferrerDomain:     nil,
				ReferrerURL:        nil,
				ProjectID:          project2.ID,
				OrganizationID:     project2.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project2.Domain,
			},
		}

		err := fixtures.InsertEventsDirect(t, tc2, events)
		require.NoError(t, err)

		time.Sleep(1 * time.Second)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project2.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project2.OrganizationID)

		response, err := tile2.FetchByReferer(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		var directData *tiles.RefererTrafficSourceData
		for i := range response.Data {
			if response.Data[i].RefererURL == "DIRECT/NONE" {
				directData = &response.Data[i]
				break
			}
		}

		require.NotNil(t, directData, "Should have DIRECT/NONE entry for missing referrers")
		assert.NotNil(t, directData.Count)
		assert.Equal(t, uint64(2), *directData.Count, "Should have 2 direct visitors")

		t.Logf("Direct traffic: %d visitors", *directData.Count)
	})

	t.Run("should order by current visits descending", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		tile3 := tiles.NewTrafficRefererSourceTile(tc3.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		now := time.Now().UTC()

		var events []fixtures.EventInsertData

		for i := 0; i < 5; i++ {
			events = append(events, fixtures.EventInsertData{
				VisitorID:          "visitor-twitter-" + string(rune(i)),
				SessionID:          "session-twitter-" + string(rune(i)),
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				ReferrerDomain:     nullable.FromString("twitter.com").Ptr(),
				ProjectID:          project3.ID,
				OrganizationID:     project3.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project3.Domain,
			})
		}

		for i := 0; i < 2; i++ {
			events = append(events, fixtures.EventInsertData{
				VisitorID:          "visitor-reddit-" + string(rune(i)),
				SessionID:          "session-reddit-" + string(rune(i)),
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				ReferrerDomain:     nullable.FromString("reddit.com").Ptr(),
				ProjectID:          project3.ID,
				OrganizationID:     project3.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project3.Domain,
			})
		}

		err := fixtures.InsertEventsDirect(t, tc3, events)
		require.NoError(t, err)

		time.Sleep(1 * time.Second)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project3.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project3.OrganizationID)

		response, err := tile3.FetchByReferer(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.GreaterOrEqual(t, len(response.Data), 2, "Should have at least 2 referrer sources")

		if len(response.Data) >= 2 {
			assert.GreaterOrEqual(t, *response.Data[0].Count, *response.Data[1].Count,
				"Results should be ordered by current visits descending")

			t.Logf("Top referrer: %s with %d visits", response.Data[0].RefererURL, *response.Data[0].Count)
			t.Logf("Second referrer: %s with %d visits", response.Data[1].RefererURL, *response.Data[1].Count)
		}
	})

	t.Run("should handle empty data gracefully", func(t *testing.T) {
		tc4 := di.NewTestContainer(t)
		defer tc4.Cleanup()

		_, project4 := fixtures.SetupAccountAndProject(t, tc4)
		tile4 := tiles.NewTrafficRefererSourceTile(tc4.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project4.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project4.OrganizationID)

		response, err := tile4.FetchByReferer(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Empty(t, response.Data, "Should have empty data for no events")
	})
}
