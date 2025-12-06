package tiles_test

import (
	"testing"
	"time"

	"zori/di"
	"zori/services/analytics/filters"
	"zori/services/analytics/tiles"
	"zori/testutil/fixtures"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMCostTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewLLMCostTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should sum LLM costs in current and previous periods", func(t *testing.T) {
		now := time.Now().UTC()
		gpt4 := "gpt-4"
		claude := "claude-3-opus"

		cost1 := 0.05
		cost2 := 0.03
		cost3 := 0.02
		costOld1 := 0.04
		costOld2 := 0.01

		generations := []fixtures.LLMGenerationInsertData{
			{
				ProjectID:    project.ID,
				GenerationID: uuid.New().String(),
				TraceID:      uuid.New().String(),
				Model:        &gpt4,
				StartTime:    now.Add(-2 * time.Hour),
				TotalCost:    &cost1,
			},
			{
				ProjectID:    project.ID,
				GenerationID: uuid.New().String(),
				TraceID:      uuid.New().String(),
				Model:        &gpt4,
				StartTime:    now.Add(-3 * time.Hour),
				TotalCost:    &cost2,
			},
			{
				ProjectID:    project.ID,
				GenerationID: uuid.New().String(),
				TraceID:      uuid.New().String(),
				Model:        &claude,
				StartTime:    now.Add(-4 * time.Hour),
				TotalCost:    &cost3,
			},
			{
				ProjectID:    project.ID,
				GenerationID: uuid.New().String(),
				TraceID:      uuid.New().String(),
				Model:        &gpt4,
				StartTime:    now.Add(-10 * 24 * time.Hour),
				TotalCost:    &costOld1,
			},
			{
				ProjectID:    project.ID,
				GenerationID: uuid.New().String(),
				TraceID:      uuid.New().String(),
				Model:        &claude,
				StartTime:    now.Add(-11 * 24 * time.Hour),
				TotalCost:    &costOld2,
			},
		}

		err := fixtures.InsertLLMGenerationsDirect(t, tc, generations)
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

		assert.InDelta(t, 0.10, response.Cost, 0.001, "Should have $0.10 total cost in current period")
		assert.InDelta(t, 0.05, response.PreviousCost, 0.001, "Should have $0.05 total cost in previous period")

		t.Logf("Current LLM cost: $%.4f, Previous: $%.4f", response.Cost, response.PreviousCost)
	})

	t.Run("should return zero when no generations exist", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		tile2 := tiles.NewLLMCostTile(tc2.ClickHouse)

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

		assert.Equal(t, float64(0), response.Cost, "Should have 0 cost when no generations")
		assert.Equal(t, float64(0), response.PreviousCost, "Should have 0 previous cost when no generations")
	})
}
