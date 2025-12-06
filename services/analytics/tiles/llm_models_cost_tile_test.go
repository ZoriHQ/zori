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

func TestLLMTopModelsCostTile_Fetch(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	tile := tiles.NewLLMTopModelsCostTile(tc.ClickHouse)

	time.Sleep(500 * time.Millisecond)

	t.Run("should return top 3 models by cost with deltas", func(t *testing.T) {
		now := time.Now().UTC()
		gpt4 := "gpt-4"
		claude := "claude-3-opus"
		gpt35 := "gpt-3.5-turbo"
		mistral := "mistral-7b"

		// Current period costs
		cost1 := 0.50 // gpt-4
		cost2 := 0.30 // gpt-4
		cost3 := 0.25 // claude
		cost4 := 0.15 // gpt-3.5
		cost5 := 0.05 // mistral (should be excluded from top 3)

		// Previous period costs
		costOld1 := 0.40 // gpt-4
		costOld2 := 0.20 // claude
		costOld3 := 0.10 // gpt-3.5

		generations := []fixtures.LLMGenerationInsertData{
			// Current period - gpt-4 (total: 0.80)
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
				Model:        &gpt35,
				StartTime:    now.Add(-5 * time.Hour),
				TotalCost:    &cost4,
			},
			{
				ProjectID:    project.ID,
				GenerationID: uuid.New().String(),
				TraceID:      uuid.New().String(),
				Model:        &mistral,
				StartTime:    now.Add(-6 * time.Hour),
				TotalCost:    &cost5,
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
			{
				ProjectID:    project.ID,
				GenerationID: uuid.New().String(),
				TraceID:      uuid.New().String(),
				Model:        &gpt35,
				StartTime:    now.Add(-12 * 24 * time.Hour),
				TotalCost:    &costOld3,
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

		assert.Len(t, response.Data, 3, "Should have top 3 models")

		assert.Equal(t, "gpt-4", response.Data[0].Model)
		assert.InDelta(t, 0.80, response.Data[0].Cost, 0.001)
		assert.InDelta(t, 0.40, response.Data[0].PreviousCost, 0.001)

		assert.Equal(t, "claude-3-opus", response.Data[1].Model)
		assert.InDelta(t, 0.25, response.Data[1].Cost, 0.001)
		assert.InDelta(t, 0.20, response.Data[1].PreviousCost, 0.001)

		assert.Equal(t, "gpt-3.5-turbo", response.Data[2].Model)
		assert.InDelta(t, 0.15, response.Data[2].Cost, 0.001)
		assert.InDelta(t, 0.10, response.Data[2].PreviousCost, 0.001)

		t.Logf("Top 3 models by cost:")
		for _, m := range response.Data {
			t.Logf("  %s: $%.4f (prev: $%.4f)", m.Model, m.Cost, m.PreviousCost)
		}
	})

	t.Run("should return empty array when no generations exist", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		tile2 := tiles.NewLLMTopModelsCostTile(tc2.ClickHouse)

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

		assert.Empty(t, response.Data, "Should have empty data when no generations")
	})

	t.Run("should handle unknown models", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		tile3 := tiles.NewLLMTopModelsCostTile(tc3.ClickHouse)

		time.Sleep(500 * time.Millisecond)

		now := time.Now().UTC()
		cost := 0.10

		generations := []fixtures.LLMGenerationInsertData{
			{
				ProjectID:    project3.ID,
				GenerationID: uuid.New().String(),
				TraceID:      uuid.New().String(),
				Model:        nil,
				StartTime:    now.Add(-2 * time.Hour),
				TotalCost:    &cost,
			},
		}

		err := fixtures.InsertLLMGenerationsDirect(t, tc3, generations)
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

		response, err := tile3.Fetch(appCtx, filter)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, "unknown", response.Data[0].Model)
	})
}
