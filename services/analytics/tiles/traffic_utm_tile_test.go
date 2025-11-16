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

func TestTrafficUTMSourceTile_FetchByUTM(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewTrafficUTMSourceTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should aggregate traffic by UTM parameters", func(t *testing.T) {
		now := time.Now().UTC()

		googleSource := "google"
		googleMedium := "cpc"
		googleCampaign := "spring_sale"

		facebookSource := "facebook"
		facebookMedium := "social"
		facebookCampaign := "awareness"

		events := []fixtures.EventInsertData{
			{
				VisitorID:          "visitor-google-1",
				SessionID:          "session-1",
				ClientTimestampUTC: now.Add(-1 * time.Hour),
				UTMSource:          &googleSource,
				UTMMedium:          &googleMedium,
				UTMCampaign:        &googleCampaign,
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
				ClientTimestampUTC: now.Add(-2 * time.Hour),
				UTMSource:          &googleSource,
				UTMMedium:          &googleMedium,
				UTMCampaign:        &googleCampaign,
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
				UTMSource:          &facebookSource,
				UTMMedium:          &facebookMedium,
				UTMCampaign:        &facebookCampaign,
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

		response, err := tile.FetchByUTM(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.NotEmpty(t, response.Data)

		var googleData *tiles.UTMTrafficSourceData
		for i := range response.Data {
			if response.Data[i].UTMSource == "google" &&
				response.Data[i].UTMMedium == "cpc" &&
				response.Data[i].UTMCampaign == "spring_sale" {
				googleData = &response.Data[i]
				break
			}
		}

		require.NotNil(t, googleData, "Google campaign data should be present")
		assert.NotNil(t, googleData.Count)
		assert.Equal(t, uint64(2), *googleData.Count)

		var facebookData *tiles.UTMTrafficSourceData
		for i := range response.Data {
			if response.Data[i].UTMSource == "facebook" &&
				response.Data[i].UTMMedium == "social" &&
				response.Data[i].UTMCampaign == "awareness" {
				facebookData = &response.Data[i]
				break
			}
		}

		require.NotNil(t, facebookData, "Facebook campaign data should be present")
		assert.NotNil(t, facebookData.Count)
		assert.Equal(t, uint64(1), *facebookData.Count)
	})
}
