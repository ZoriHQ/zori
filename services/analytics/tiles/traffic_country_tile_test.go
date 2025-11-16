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

func TestTrafficCountrySourceTile_FetchByCountry(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewTrafficCountrySourceTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should aggregate traffic by country", func(t *testing.T) {
		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-us-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				LocationCountryISO: nullable.FromString("US").Ptr(),
				LocationCity:       nullable.FromString("New York").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-us-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-3 * time.Hour),
				LocationCountryISO: nullable.FromString("US").Ptr(),
				LocationCity:       nullable.FromString("San Francisco").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-us-3",
				SessionID:          "session-3",
				ClientTimestampUTC: now.Add(-4 * time.Hour),
				LocationCountryISO: nullable.FromString("US").Ptr(),
				LocationCity:       nullable.FromString("Austin").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-uk-1",
				SessionID:          "session-4",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				LocationCountryISO: nullable.FromString("GB").Ptr(),
				LocationCity:       nullable.FromString("London").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-uk-2",
				SessionID:          "session-5",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				LocationCountryISO: nullable.FromString("GB").Ptr(),
				LocationCity:       nullable.FromString("Manchester").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-us-old",
				SessionID:          "session-6",
				ClientTimestampUTC: now.Add(-10 * 24 * time.Hour),
				LocationCountryISO: nullable.FromString("US").Ptr(),
				LocationCity:       nullable.FromString("Chicago").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-uk-old-1",
				SessionID:          "session-7",
				ClientTimestampUTC: now.Add(-11 * 24 * time.Hour),
				LocationCountryISO: nullable.FromString("GB").Ptr(),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project.Domain,
			},
			{
				VisitorID:          "visitor-uk-old-2",
				SessionID:          "session-8",
				ClientTimestampUTC: now.Add(-12 * 24 * time.Hour),
				LocationCountryISO: nullable.FromString("GB").Ptr(),
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

		assert.NotEmpty(t, response.Data, "Should have country traffic data")

		assert.Equal(t, 2, len(response.Data), "Should have exactly 2 countries (US and GB)")

		countrySeen := make(map[string]bool)
		var usData, ukData *tiles.CountryTrafficSourceData

		for i := range response.Data {
			country := response.Data[i].Country

			assert.False(t, countrySeen[country], "Country %s should not appear more than once", country)
			countrySeen[country] = true

			assert.Contains(t, []string{"US", "GB"}, country, "Country should be either US or GB")

			if country == "US" {
				usData = response.Data[i]
			}
			if country == "GB" {
				ukData = response.Data[i]
			}
		}

		require.NotNil(t, usData, "Should have US country data")
		assert.Equal(t, "US", usData.Country, "Country code should be US")
		assert.Equal(t, uint64(3), usData.Count, "US should have 3 current visitors")
		assert.Equal(t, uint64(1), usData.PreviousCount, "US should have 1 previous visitor")

		require.NotNil(t, ukData, "Should have UK country data")
		assert.Equal(t, "GB", ukData.Country, "Country code should be GB")
		assert.Equal(t, uint64(2), ukData.Count, "UK should have 2 current visitors")
		assert.Equal(t, uint64(2), ukData.PreviousCount, "UK should have 2 previous visitors")

		t.Logf("US: current=%d, previous=%d", usData.Count, usData.PreviousCount)
		t.Logf("UK: current=%d, previous=%d", ukData.Count, ukData.PreviousCount)
	})

	t.Run("should handle UNKNOWN for missing country", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		tile2 := tiles.NewTrafficCountrySourceTile(tc2.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-unknown-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				LocationCountryISO: nil,
				ProjectID:          project2.ID,
				OrganizationID:     project2.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project2.Domain,
			},
			{
				VisitorID:          "visitor-unknown-2",
				SessionID:          "session-2",
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				LocationCountryISO: nil,
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

		response, err := tile2.FetchByCountry(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Equal(t, 1, len(response.Data), "Should have exactly 1 country entry (UNKNOWN)")

		countrySeen := make(map[string]bool)
		var unknownData *tiles.CountryTrafficSourceData

		for i := range response.Data {
			country := response.Data[i].Country

			assert.False(t, countrySeen[country], "Country %s should not appear more than once", country)
			countrySeen[country] = true

			assert.Equal(t, "UNKNOWN", country, "Country should be UNKNOWN for missing country data")

			if country == "UNKNOWN" {
				unknownData = response.Data[i]
			}
		}

		require.NotNil(t, unknownData, "Should have UNKNOWN entry for missing countries")
		assert.Equal(t, "UNKNOWN", unknownData.Country, "Country code should be UNKNOWN")
		assert.Equal(t, uint64(2), unknownData.Count, "Should have 2 unknown country visitors")

		t.Logf("Unknown country traffic: %d visitors", unknownData.Count)
	})

	t.Run("should filter out zero-count countries with HAVING clause", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		tile3 := tiles.NewTrafficCountrySourceTile(tc3.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		now := time.Now().UTC()

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-old",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-10 * 24 * time.Hour),
				LocationCountryISO: nullable.FromString("DE").Ptr(),
				ProjectID:          project3.ID,
				OrganizationID:     project3.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project3.Domain,
			},
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

		response, err := tile3.FetchByCountry(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		for i := range response.Data {
			assert.NotEqual(t, "DE", response.Data[i].Country,
				"Should not include countries with 0 current visits")
			assert.Greater(t, response.Data[i].Count, uint64(0),
				"All returned countries should have count > 0")
		}

		t.Logf("Correctly filtered out countries with zero current visits")
	})

	t.Run("should order by current visits descending", func(t *testing.T) {
		tc4 := di.NewTestContainer(t)
		defer tc4.Cleanup()

		_, project4 := fixtures.SetupAccountAndProject(t, tc4)
		tile4 := tiles.NewTrafficCountrySourceTile(tc4.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		now := time.Now().UTC()

		var events []fixtures.EventInsertData

		for i := 0; i < 5; i++ {
			events = append(events, fixtures.EventInsertData{
				VisitorID:          "visitor-fr-" + string(rune(i)),
				SessionID:          "session-fr-" + string(rune(i)),
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				LocationCountryISO: nullable.FromString("FR").Ptr(),
				ProjectID:          project4.ID,
				OrganizationID:     project4.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project4.Domain,
			})
		}

		for i := 0; i < 2; i++ {
			events = append(events, fixtures.EventInsertData{
				VisitorID:          "visitor-ca-" + string(rune(i)),
				SessionID:          "session-ca-" + string(rune(i)),
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				LocationCountryISO: nullable.FromString("CA").Ptr(),
				ProjectID:          project4.ID,
				OrganizationID:     project4.OrganizationID,
				UserAgent:          "Mozilla/5.0",
				IP:                 "127.0.0.1",
				PageURL:            "/",
				PagePath:           "/",
				Host:               project4.Domain,
			})
		}

		err := fixtures.InsertEventsDirect(t, tc4, events)
		require.NoError(t, err)

		time.Sleep(1 * time.Second)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project4.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project4.OrganizationID)

		response, err := tile4.FetchByCountry(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.GreaterOrEqual(t, len(response.Data), 2, "Should have at least 2 countries")
		assert.Equal(t, 2, len(response.Data), "Should have exactly 2 countries (FR and CA)")

		countrySeen := make(map[string]bool)
		for i := range response.Data {
			country := response.Data[i].Country
			assert.False(t, countrySeen[country], "Country %s should not appear more than once", country)
			countrySeen[country] = true

			assert.Contains(t, []string{"FR", "CA"}, country, "Country should be either FR or CA")
		}

		if len(response.Data) >= 2 {
			assert.GreaterOrEqual(t, response.Data[0].Count, response.Data[1].Count,
				"Results should be ordered by current visits descending")

			t.Logf("Top country: %s with %d visits", response.Data[0].Country, response.Data[0].Count)
			t.Logf("Second country: %s with %d visits", response.Data[1].Country, response.Data[1].Count)
		}
	})

	t.Run("should handle empty data gracefully", func(t *testing.T) {
		tc5 := di.NewTestContainer(t)
		defer tc5.Cleanup()

		_, project5 := fixtures.SetupAccountAndProject(t, tc5)
		tile5 := tiles.NewTrafficCountrySourceTile(tc5.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID:      project5.ID,
			TimeBoundaries: filters.TimeBoundariesLastWeek,
			TimeRange:      timeRange,
		}

		appCtx := fixtures.NewTestCtxWithOrg(project5.OrganizationID)

		response, err := tile5.FetchByCountry(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Empty(t, response.Data, "Should have empty data for no events")
	})
}
