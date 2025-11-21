package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zori/di"
	appCtx "zori/internal/ctx"
	"zori/services/analytics/filters"
	analyticsTypes "zori/services/analytics/types"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalytics_TopVisitors_UnidentifiedVisitorPayments(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	analyticsData := tc.AnalyticsData
	ctx := context.Background()
	appContext := &appCtx.Ctx{Context: ctx}

	time.Sleep(1 * time.Second)

	t.Run("should show revenue for unidentified visitors with payments", func(t *testing.T) {
		identifiedVisitor1 := "visitor-identified-1"
		identifiedVisitor2 := "visitor-identified-2"
		unidentifiedVisitor := "visitor-anonymous-paying"

		event1 := fixtures.NewEventBuilder().
			WithVisitorID(identifiedVisitor1).
			WithSessionID("session-1").
			WithIdentity("user-1", "", "user1@example.com").
			WithPageURL("/products").
			WithHost(project.Domain).
			WithUTMSource("google").
			Build()

		event2 := fixtures.NewEventBuilder().
			WithVisitorID(identifiedVisitor2).
			WithSessionID("session-2").
			WithIdentity("user-2", "", "user2@example.com").
			WithPageURL("/pricing").
			WithHost(project.Domain).
			WithUTMSource("facebook").
			Build()

		event3 := fixtures.NewEventBuilder().
			WithVisitorID(unidentifiedVisitor).
			WithSessionID("session-3").
			WithPageURL("/landing").
			WithHost(project.Domain).
			WithUTMSource("twitter").
			Build()

		err := fixtures.SendEventToTestServer(t, tc, project, event1)
		require.NoError(t, err)
		err = fixtures.SendEventToTestServer(t, tc, project, event2)
		require.NoError(t, err)
		err = fixtures.SendEventToTestServer(t, tc, project, event3)
		require.NoError(t, err)

		_, err = fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 3, 15*time.Second)
		require.NoError(t, err, "Events should appear in ClickHouse")

		time.Sleep(2 * time.Second)

		payment1 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(identifiedVisitor1).
			WithAmount(10000). // $100
			WithProductName("Product A").
			Build()

		payment2 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(identifiedVisitor2).
			WithAmount(15000). // $150
			WithProductName("Product B").
			Build()

		payment3 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(unidentifiedVisitor).
			WithAmount(8000). // $80
			WithProductName("Product C").
			Build()

		err = fixtures.SendPaymentToNATS(t, tc, payment1)
		require.NoError(t, err)
		err = fixtures.SendPaymentToNATS(t, tc, payment2)
		require.NoError(t, err)
		err = fixtures.SendPaymentToNATS(t, tc, payment3)
		require.NoError(t, err)

		succeededStatus := "succeeded"
		payments, err := fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			PaymentStatus: &succeededStatus,
		}, 3, 15*time.Second)
		require.NoError(t, err, "All 3 payments should appear in ClickHouse")
		assert.Len(t, payments, 3, "Should have stored all 3 payments")

		time.Sleep(3 * time.Second)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID: project.ID,
			TimeRange: timeRange,
			Limit:     100,
		}

		visitors, err := analyticsData.GetTopVisitors(appContext, filter)
		require.NoError(t, err, "Should successfully query top visitors")
		require.GreaterOrEqual(t, len(visitors), 3, "Should have at least 3 visitors")
		t.Logf("Found %d total visitors", len(visitors))

		var visitor1, visitor2, visitor3 *analyticsTypes.TopVisitor
		for i := range visitors {
			if visitors[i].UserID != nil && *visitors[i].UserID == "user-1" {
				visitor1 = &visitors[i]
			} else if visitors[i].UserID != nil && *visitors[i].UserID == "user-2" {
				visitor2 = &visitors[i]
			} else {
				// Check if this is our unidentified visitor by visitor_id
				for _, vid := range visitors[i].VisitorIDs {
					if vid == unidentifiedVisitor {
						visitor3 = &visitors[i]
						break
					}
				}
			}
		}

		require.NotNil(t, visitor1, "Should find identified visitor 1")
		assert.Equal(t, uint64(1), visitor1.DistinctPayments, "Visitor 1 should have 1 payment")
		assert.Equal(t, 100.0, visitor1.TotalRevenue, "Visitor 1 should have $100 revenue")
		t.Logf("✓ Identified visitor 1: %d payments, $%.2f revenue",
			visitor1.DistinctPayments, visitor1.TotalRevenue)

		require.NotNil(t, visitor2, "Should find identified visitor 2")
		assert.Equal(t, uint64(1), visitor2.DistinctPayments, "Visitor 2 should have 1 payment")
		assert.Equal(t, 150.0, visitor2.TotalRevenue, "Visitor 2 should have $150 revenue")
		t.Logf("✓ Identified visitor 2: %d payments, $%.2f revenue",
			visitor2.DistinctPayments, visitor2.TotalRevenue)

		require.NotNil(t, visitor3, "Should find unidentified visitor (bug fix test)")
		assert.Equal(t, uint64(1), visitor3.DistinctPayments,
			"BUG FIX: Unidentified visitor should have 1 payment. "+
				"If this is 0, payments for unidentified visitors are not being queried.")
		assert.Equal(t, 80.0, visitor3.TotalRevenue,
			"BUG FIX: Unidentified visitor should have $80 revenue. "+
				"If this is 0, the payment query is missing unidentified visitor IDs.")

		assert.Nil(t, visitor3.UserID, "Unidentified visitor should have no user_id")
		assert.Nil(t, visitor3.ExternalID, "Unidentified visitor should have no external_id")
		assert.Nil(t, visitor3.Email, "Unidentified visitor should have no email")
		assert.False(t, visitor3.IsGrouped, "Unidentified visitor should not be grouped")

		t.Logf("✓ Unidentified visitor: %d payments, $%.2f revenue (BUG FIXED)",
			visitor3.DistinctPayments, visitor3.TotalRevenue)
	})

	t.Run("should handle mix of identified and unidentified visitors correctly", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		time.Sleep(1 * time.Second)

		unidentified1 := "unidentified-payer-1"
		unidentified2 := "unidentified-payer-2"
		identified1 := "identified-payer-1"

		for i := 0; i < 5; i++ {
			event := fixtures.NewEventBuilder().
				WithVisitorID(unidentified1).
				WithSessionID(fmt.Sprintf("session-u1-%d", i)).
				WithPageURL(fmt.Sprintf("/page-%d", i)).
				WithHost(project2.Domain).
				Build()
			err := fixtures.SendEventToTestServer(t, tc2, project2, event)
			require.NoError(t, err)
		}

		for i := 0; i < 3; i++ {
			event := fixtures.NewEventBuilder().
				WithVisitorID(unidentified2).
				WithSessionID(fmt.Sprintf("session-u2-%d", i)).
				WithPageURL(fmt.Sprintf("/page-%d", i)).
				WithHost(project2.Domain).
				Build()
			err := fixtures.SendEventToTestServer(t, tc2, project2, event)
			require.NoError(t, err)
		}

		for i := 0; i < 4; i++ {
			event := fixtures.NewEventBuilder().
				WithVisitorID(identified1).
				WithSessionID(fmt.Sprintf("session-i1-%d", i)).
				WithIdentity("user-identified", "ext-123", "identified@example.com").
				WithPageURL(fmt.Sprintf("/page-%d", i)).
				WithHost(project2.Domain).
				Build()
			err := fixtures.SendEventToTestServer(t, tc2, project2, event)
			require.NoError(t, err)
		}

		_, err := fixtures.WaitForEvents(t, tc2, fixtures.QueryEventsOptions{
			ProjectID: &project2.ID,
		}, 12, 20*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		payment1 := fixtures.NewPaymentBuilder(project2.ID, project2.OrganizationID).
			WithVisitorID(unidentified1).
			WithAmount(5000).
			Build()

		payment2 := fixtures.NewPaymentBuilder(project2.ID, project2.OrganizationID).
			WithVisitorID(unidentified2).
			WithAmount(7500).
			Build()

		payment3 := fixtures.NewPaymentBuilder(project2.ID, project2.OrganizationID).
			WithVisitorID(identified1).
			WithAmount(12000).
			Build()

		err = fixtures.SendPaymentToNATS(t, tc2, payment1)
		require.NoError(t, err)
		err = fixtures.SendPaymentToNATS(t, tc2, payment2)
		require.NoError(t, err)
		err = fixtures.SendPaymentToNATS(t, tc2, payment3)
		require.NoError(t, err)

		succeededStatus := "succeeded"
		_, err = fixtures.WaitForPayments(t, tc2, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project2.ID,
			PaymentStatus: &succeededStatus,
		}, 3, 15*time.Second)
		require.NoError(t, err)

		time.Sleep(3 * time.Second)

		timeRange, err := filters.ValidateTimeRange(filters.TimeBoundariesLastWeek)
		require.NoError(t, err)

		filter := &filters.SectionFilter{
			ProjectID: project2.ID,
			TimeRange: timeRange,
			Limit:     100,
		}

		visitors, err := tc2.AnalyticsData.GetTopVisitors(appContext, filter)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(visitors), 3, "Should return at least 3 visitors")

		visitorsWithRevenue := 0
		totalRevenue := 0.0
		for _, v := range visitors {
			if v.TotalRevenue > 0 {
				visitorsWithRevenue++
				totalRevenue += v.TotalRevenue

				identity := "unidentified"
				if v.UserID != nil {
					identity = *v.UserID
				}
				t.Logf("Visitor %s: %d events, $%.2f revenue, %d payments",
					identity, v.EventCount, v.TotalRevenue, v.DistinctPayments)
			}
		}

		assert.Equal(t, 3, visitorsWithRevenue,
			"All 3 visitors (2 unidentified + 1 identified) should show revenue")
		assert.InDelta(t, 245.0, totalRevenue, 5.0,
			"Total revenue should be $245 ($50 + $75 + $120)")

		t.Logf("✓ All %d paying visitors show revenue correctly (identified and unidentified)",
			visitorsWithRevenue)
	})
}
