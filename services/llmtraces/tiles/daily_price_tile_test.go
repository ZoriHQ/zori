package tiles_test

import (
	"testing"
	"time"

	"zori/di"
	"zori/services/analytics/filters"
	"zori/services/llmtraces/tiles"
	"zori/testutil/fixtures"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyPriceTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewDailyPriceTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should calculate total cost correctly", func(t *testing.T) {
		now := time.Now().UTC()

		traceData := []fixtures.LLMTraceInsertData{
			{
				CustomerID:         fixtures.StrPtr("customer-1"),
				Provider:           "openai",
				Model:              "gpt-4o",
				InputTokens:        100,
				OutputTokens:       200,
				InputCost:          decimal.NewFromFloat(0.001),
				OutputCost:         decimal.NewFromFloat(0.002),
				TotalCost:          decimal.NewFromFloat(0.003),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				ClientTimestampUTC: now.Add(-1 * time.Hour),
			},
			{
				CustomerID:         fixtures.StrPtr("customer-2"),
				Provider:           "anthropic",
				Model:              "claude-3-sonnet",
				InputTokens:        150,
				OutputTokens:       300,
				InputCost:          decimal.NewFromFloat(0.002),
				OutputCost:         decimal.NewFromFloat(0.004),
				TotalCost:          decimal.NewFromFloat(0.006),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				ClientTimestampUTC: now.Add(-2 * time.Hour),
			},
			{
				CustomerID:         fixtures.StrPtr("customer-1"),
				Provider:           "openai",
				Model:              "gpt-4o",
				InputTokens:        50,
				OutputTokens:       100,
				InputCost:          decimal.NewFromFloat(0.0005),
				OutputCost:         decimal.NewFromFloat(0.001),
				TotalCost:          decimal.NewFromFloat(0.0015),
				ProjectID:          project.ID,
				OrganizationID:     project.OrganizationID,
				ClientTimestampUTC: now.Add(-10 * 24 * time.Hour),
			},
		}

		err := fixtures.InsertLLMTracesDirect(t, tc, traceData)
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

		expectedTotalCost := decimal.NewFromFloat(0.009)
		assert.True(t, response.TotalCost.Equal(expectedTotalCost), "Expected total cost %s, got %s", expectedTotalCost, response.TotalCost)
		assert.Equal(t, uint64(2), response.TraceCount, "Should have 2 traces in current period")
		assert.Equal(t, uint64(250), response.TotalInputTokens)
		assert.Equal(t, uint64(500), response.TotalOutputTokens)

		t.Logf("Total cost: %s, Previous: %s, Trace count: %d", response.TotalCost, response.PreviousTotalCost, response.TraceCount)
	})

	t.Run("should return zero when no traces exist", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		tile2 := tiles.NewDailyPriceTile(tc2.ClickHouse)

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

		assert.True(t, response.TotalCost.Equal(decimal.Zero), "Should have 0 total cost when no traces")
		assert.Equal(t, uint64(0), response.TraceCount, "Should have 0 traces")
	})
}
