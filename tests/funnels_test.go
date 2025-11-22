package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"zori/di"
	appCtx "zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/services/funnels/types"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunnels_SequentialFunnel_BasicConversion(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	ctx := context.Background()
	appContext := &appCtx.Ctx{Context: ctx}

	time.Sleep(1 * time.Second)

	t.Run("should track conversion through sequential funnel steps", func(t *testing.T) {
		visitor1 := "visitor-convert-all"
		visitor2 := "visitor-convert-partial"
		visitor3 := "visitor-no-convert"

		sendFunnelEvents(t, tc, project, visitor1, []string{"/product/123", "/cart", "/checkout", "/thank-you"})
		sendFunnelEvents(t, tc, project, visitor2, []string{"/product/456", "/cart"})
		sendFunnelEvents(t, tc, project, visitor3, []string{"/home", "/about"})

		_, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 8, 15*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		step1Condition, _ := json.Marshal(types.StepCondition{
			Type:        types.ConditionTypePageview,
			PagePattern: strPtr("/product/.*"),
		})
		step2Condition, _ := json.Marshal(types.StepCondition{
			Type:     types.ConditionTypePageview,
			PagePath: strPtr("/cart"),
		})
		step3Condition, _ := json.Marshal(types.StepCondition{
			Type:     types.ConditionTypePageview,
			PagePath: strPtr("/checkout"),
		})
		step4Condition, _ := json.Marshal(types.StepCondition{
			Type:     types.ConditionTypePageview,
			PagePath: strPtr("/thank-you"),
		})

		funnel := &models.Funnel{
			ProjectID:      project.ID,
			OrganizationID: project.OrganizationID,
			Name:           "Test Checkout Funnel",
			FunnelType:     models.FunnelTypeSequential,
			WindowSeconds:  3600,
			IsStrict:       false,
			Steps: []*models.FunnelStep{
				{StepOrder: 1, Name: "View Product", Condition: step1Condition},
				{StepOrder: 2, Name: "View Cart", Condition: step2Condition},
				{StepOrder: 3, Name: "Checkout", Condition: step3Condition},
				{StepOrder: 4, Name: "Thank You", Condition: step4Condition},
			},
		}

		err = tc.FunnelRepository.CreateFunnel(ctx, funnel)
		require.NoError(t, err)

		for _, step := range funnel.Steps {
			step.FunnelID = funnel.ID
		}
		err = tc.FunnelRepository.CreateFunnelSteps(ctx, funnel.Steps)
		require.NoError(t, err)

		createdFunnel, err := tc.FunnelRepository.GetFunnel(ctx, funnel.ID, project.OrganizationID)
		require.NoError(t, err)

		startTime := time.Now().Add(-24 * time.Hour)
		result, err := tc.FunnelAnalyticsData.AnalyzeSequentialFunnel(appContext, createdFunnel, startTime)
		require.NoError(t, err)

		assert.Equal(t, funnel.ID, result.FunnelID)
		assert.Equal(t, "Test Checkout Funnel", result.FunnelName)
		assert.Len(t, result.Steps, 4)

		t.Logf("Funnel Analysis Results:")
		t.Logf("  Total Visitors: %d", result.TotalVisitors)
		t.Logf("  Converted: %d", result.ConvertedVisitors)
		t.Logf("  Overall Conversion: %.2f%%", result.OverallConversion*100)

		for _, step := range result.Steps {
			t.Logf("  Step %d (%s): %d visitors, %.2f%% conversion, %.2f%% dropoff",
				step.StepOrder, step.Name, step.Visitors, step.ConversionRate*100, step.DropoffRate*100)
		}

		assert.GreaterOrEqual(t, result.TotalVisitors, uint64(2), "At least 2 visitors should enter funnel")
		assert.GreaterOrEqual(t, result.ConvertedVisitors, uint64(1), "At least 1 visitor should complete funnel")
	})
}

func TestFunnels_SequentialFunnel_WithEventConditions(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	ctx := context.Background()
	appContext := &appCtx.Ctx{Context: ctx}

	time.Sleep(1 * time.Second)

	t.Run("should track custom events in funnel", func(t *testing.T) {
		visitor1 := "visitor-event-funnel-1"

		event1 := fixtures.NewEventBuilder().
			WithVisitorID(visitor1).
			WithSessionID("session-ef-1").
			WithEventName("signup_started").
			WithPageURL("/signup").
			WithHost(project.Domain).
			Build()

		event2 := fixtures.NewEventBuilder().
			WithVisitorID(visitor1).
			WithSessionID("session-ef-1").
			WithEventName("signup_completed").
			WithPageURL("/welcome").
			WithHost(project.Domain).
			Build()

		err := fixtures.SendEventToTestServer(t, tc, project, event1)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		err = fixtures.SendEventToTestServer(t, tc, project, event2)
		require.NoError(t, err)

		_, err = fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 2, 15*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		step1Condition, _ := json.Marshal(types.StepCondition{
			Type:      types.ConditionTypeEvent,
			EventName: strPtr("signup_started"),
		})
		step2Condition, _ := json.Marshal(types.StepCondition{
			Type:      types.ConditionTypeEvent,
			EventName: strPtr("signup_completed"),
		})

		funnel := &models.Funnel{
			ProjectID:      project.ID,
			OrganizationID: project.OrganizationID,
			Name:           "Signup Event Funnel",
			FunnelType:     models.FunnelTypeSequential,
			WindowSeconds:  3600,
			IsStrict:       false,
			Steps: []*models.FunnelStep{
				{StepOrder: 1, Name: "Signup Started", Condition: step1Condition},
				{StepOrder: 2, Name: "Signup Completed", Condition: step2Condition},
			},
		}

		err = tc.FunnelRepository.CreateFunnel(ctx, funnel)
		require.NoError(t, err)

		for _, step := range funnel.Steps {
			step.FunnelID = funnel.ID
		}
		err = tc.FunnelRepository.CreateFunnelSteps(ctx, funnel.Steps)
		require.NoError(t, err)

		createdFunnel, err := tc.FunnelRepository.GetFunnel(ctx, funnel.ID, project.OrganizationID)
		require.NoError(t, err)

		startTime := time.Now().Add(-24 * time.Hour)
		result, err := tc.FunnelAnalyticsData.AnalyzeSequentialFunnel(appContext, createdFunnel, startTime)
		require.NoError(t, err)

		t.Logf("Event Funnel Results:")
		for _, step := range result.Steps {
			t.Logf("  Step %d (%s): %d visitors", step.StepOrder, step.Name, step.Visitors)
		}

		assert.GreaterOrEqual(t, result.Steps[0].Visitors, uint64(1), "At least 1 visitor should start signup")
		assert.GreaterOrEqual(t, result.ConvertedVisitors, uint64(1), "At least 1 visitor should complete signup")
	})
}

func TestFunnels_CRUD_Operations(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	ctx := context.Background()

	t.Run("should create, read, update, and delete funnel", func(t *testing.T) {
		step1Condition, _ := json.Marshal(types.StepCondition{
			Type:     types.ConditionTypePageview,
			PagePath: strPtr("/landing"),
		})

		funnel := &models.Funnel{
			ProjectID:      project.ID,
			OrganizationID: project.OrganizationID,
			Name:           "CRUD Test Funnel",
			FunnelType:     models.FunnelTypeSequential,
			WindowSeconds:  3600,
		}

		err := tc.FunnelRepository.CreateFunnel(ctx, funnel)
		require.NoError(t, err)
		assert.NotEmpty(t, funnel.ID)

		steps := []*models.FunnelStep{
			{FunnelID: funnel.ID, StepOrder: 1, Name: "Landing", Condition: step1Condition},
		}
		err = tc.FunnelRepository.CreateFunnelSteps(ctx, steps)
		require.NoError(t, err)

		retrieved, err := tc.FunnelRepository.GetFunnel(ctx, funnel.ID, project.OrganizationID)
		require.NoError(t, err)
		assert.Equal(t, "CRUD Test Funnel", retrieved.Name)
		assert.Len(t, retrieved.Steps, 1)

		funnels, err := tc.FunnelRepository.ListFunnelsByProject(ctx, project.ID, project.OrganizationID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(funnels), 1)

		updates := map[string]interface{}{
			"name":           "Updated Funnel Name",
			"window_seconds": 7200,
		}
		updated, err := tc.FunnelRepository.UpdateFunnel(ctx, funnel.ID, project.OrganizationID, updates)
		require.NoError(t, err)
		assert.Equal(t, "Updated Funnel Name", updated.Name)
		assert.Equal(t, 7200, updated.WindowSeconds)

		err = tc.FunnelRepository.DeleteFunnel(ctx, funnel.ID, project.OrganizationID)
		require.NoError(t, err)

		exists, err := tc.FunnelRepository.FunnelExists(ctx, funnel.ID, project.OrganizationID)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestFunnels_DepthFunnel(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	ctx := context.Background()
	appContext := &appCtx.Ctx{Context: ctx}

	time.Sleep(1 * time.Second)

	t.Run("should track page depth from entry point", func(t *testing.T) {
		visitor1 := "visitor-depth-3pages"
		visitor2 := "visitor-depth-5pages"

		sendFunnelEvents(t, tc, project, visitor1, []string{"/pricing", "/features", "/about"})
		sendFunnelEvents(t, tc, project, visitor2, []string{"/pricing", "/demo", "/signup", "/welcome", "/dashboard"})

		_, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 8, 15*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		depthConfig := types.DepthConfig{
			EntryCondition: types.StepCondition{
				Type:     types.ConditionTypePageview,
				PagePath: strPtr("/pricing"),
			},
			CountEvent: "$pageview",
			MaxDepth:   10,
			Scope:      "session",
		}
		depthConfigJSON, _ := json.Marshal(depthConfig)

		funnel := &models.Funnel{
			ProjectID:      project.ID,
			OrganizationID: project.OrganizationID,
			Name:           "Pricing Page Engagement",
			FunnelType:     models.FunnelTypeDepth,
			DepthConfig:    depthConfigJSON,
		}

		err = tc.FunnelRepository.CreateFunnel(ctx, funnel)
		require.NoError(t, err)

		createdFunnel, err := tc.FunnelRepository.GetFunnel(ctx, funnel.ID, project.OrganizationID)
		require.NoError(t, err)

		startTime := time.Now().Add(-24 * time.Hour)
		result, err := tc.FunnelAnalyticsData.AnalyzeDepthFunnel(appContext, createdFunnel, startTime)
		require.NoError(t, err)

		t.Logf("Depth Funnel Results:")
		t.Logf("  Total Sessions: %d", result.TotalSessions)
		for _, depth := range result.DepthDistribution {
			t.Logf("  Depth %d: %d sessions", depth.Depth, depth.Visitors)
		}

		assert.Equal(t, funnel.ID, result.FunnelID)
		assert.GreaterOrEqual(t, result.TotalSessions, uint64(1), "Should have at least 1 session")
	})
}

func sendFunnelEvents(t *testing.T, tc *di.TestContainer, project *fixtures.ProjectFixture, visitorID string, pages []string) {
	t.Helper()

	sessionID := "session-" + visitorID

	for i, page := range pages {
		event := fixtures.NewEventBuilder().
			WithVisitorID(visitorID).
			WithSessionID(sessionID).
			WithPageURL(page).
			WithHost(project.Domain).
			Build()

		err := fixtures.SendEventToTestServer(t, tc, project, event)
		require.NoError(t, err)

		if i < len(pages)-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func strPtr(s string) *string {
	return &s
}
