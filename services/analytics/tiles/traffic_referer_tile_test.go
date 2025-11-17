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

		// Verify exactly 2 referrer sources and no duplicates
		assert.Equal(t, 2, len(response.Data), "Should have exactly 2 referrer sources (google.com and facebook.com)")

		// Check for duplicates
		refererSeen := make(map[string]bool)
		var googleData, facebookData *tiles.RefererTrafficSourceData

		for i := range response.Data {
			referer := response.Data[i].RefererURL

			// Check for duplicates
			assert.False(t, refererSeen[referer], "Referer %s should not appear more than once", referer)
			refererSeen[referer] = true

			// Verify only expected referers
			assert.Contains(t, []string{"google.com", "facebook.com"}, referer, "Referer should be either google.com or facebook.com")

			if referer == "google.com" {
				googleData = &response.Data[i]
			}
			if referer == "facebook.com" {
				facebookData = &response.Data[i]
			}
		}

		require.NotNil(t, googleData, "Should have Google referrer data")
		assert.Equal(t, "google.com", googleData.RefererURL, "Referer URL should be google.com")
		assert.NotNil(t, googleData.Count, "Google current count should not be nil")
		assert.Equal(t, uint64(3), *googleData.Count, "Google should have 3 current visitors")
		assert.NotNil(t, googleData.PreviousCount, "Google previous count should not be nil")
		assert.Equal(t, uint64(1), *googleData.PreviousCount, "Google should have 1 previous visitor")

		require.NotNil(t, facebookData, "Should have Facebook referrer data")
		assert.Equal(t, "facebook.com", facebookData.RefererURL, "Referer URL should be facebook.com")
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

		// Should have exactly 1 DIRECT/NONE entry
		assert.Equal(t, 1, len(response.Data), "Should have exactly 1 referrer entry (DIRECT/NONE)")

		// Check for duplicates
		refererSeen := make(map[string]bool)
		var directData *tiles.RefererTrafficSourceData

		for i := range response.Data {
			referer := response.Data[i].RefererURL

			// Check for duplicates
			assert.False(t, refererSeen[referer], "Referer %s should not appear more than once", referer)
			refererSeen[referer] = true

			// Should only be DIRECT/NONE
			assert.Equal(t, "DIRECT/NONE", referer, "Referer should be DIRECT/NONE for missing referrer data")

			if referer == "DIRECT/NONE" {
				directData = &response.Data[i]
			}
		}

		require.NotNil(t, directData, "Should have DIRECT/NONE entry for missing referrers")
		assert.Equal(t, "DIRECT/NONE", directData.RefererURL, "Referer URL should be DIRECT/NONE")
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

	t.Run("should not include referers outside the time window", func(t *testing.T) {
		tc5 := di.NewTestContainer(t)
		defer tc5.Cleanup()

		_, project5 := fixtures.SetupAccountAndProject(t, tc5)
		tile5 := tiles.NewTrafficRefererSourceTile(tc5.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		now := time.Now().UTC()

		// Create events:
		// - linkedin.com only outside the time window (20+ days ago)
		// - twitter.com only in the current period (last 2 hours)
		// - Old event from 30 days ago should be completely excluded
		events := []fixtures.EventInsertData{
			// LinkedIn - outside the 14-day window
			{
				VisitorID:          "visitor-linkedin-1",
				SessionID:          "session-linkedin-1",
				ClientTimestampUTC: now.Add(-20 * 24 * time.Hour), // 20 days ago
				ReferrerDomain:     nullable.FromString("linkedin.com").Ptr(),
				ProjectID:          project5.ID,
				OrganizationID:     project5.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project5.Domain,
			},
			{
				VisitorID:          "visitor-linkedin-2",
				SessionID:          "session-linkedin-2",
				ClientTimestampUTC: now.Add(-25 * 24 * time.Hour), // 25 days ago
				ReferrerDomain:     nullable.FromString("linkedin.com").Ptr(),
				ProjectID:          project5.ID,
				OrganizationID:     project5.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project5.Domain,
			},
			// Twitter - only in current period
			{
				VisitorID:          "visitor-twitter-1",
				SessionID:          "session-twitter-1",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				ReferrerDomain:     nullable.FromString("twitter.com").Ptr(),
				ProjectID:          project5.ID,
				OrganizationID:     project5.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project5.Domain,
			},
			// Very old event - should be completely excluded
			{
				VisitorID:          "visitor-old",
				SessionID:          "session-old",
				ClientTimestampUTC: now.Add(-30 * 24 * time.Hour), // 30 days ago
				ReferrerDomain:     nullable.FromString("ancient-site.com").Ptr(),
				ProjectID:          project5.ID,
				OrganizationID:     project5.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project5.Domain,
			},
		}

		err := fixtures.InsertEventsDirect(t, tc5, events)
		require.NoError(t, err)

		time.Sleep(1 * time.Second)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project5.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project5.OrganizationID)

		response, err := tile5.FetchByReferer(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Should only have twitter.com (in current period)
		// LinkedIn is outside the 14-day window (last_7_days = current 7 days + previous 7 days = 14 days total)
		// ancient-site.com is way outside the window
		assert.Equal(t, 1, len(response.Data), "Should only have referers within the time window")

		// Verify it's twitter with correct counts
		twitterData := response.Data[0]
		assert.Equal(t, "twitter.com", twitterData.RefererURL)
		assert.NotNil(t, twitterData.Count)
		assert.Equal(t, uint64(1), *twitterData.Count, "Twitter should have 1 current visitor")
		assert.NotNil(t, twitterData.PreviousCount)
		assert.Equal(t, uint64(0), *twitterData.PreviousCount, "Twitter should have 0 previous visitors")

		t.Logf("Twitter: current=%d, previous=%d", *twitterData.Count, *twitterData.PreviousCount)
	})
}
