package main

import (
	"context"
	"testing"
	"time"

	"zori/di"
	revenueTypes "zori/services/revenue/types"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevenueAttribution_EndToEnd(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)

	// Get the revenue data service
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	// Give processors time to initialize
	time.Sleep(200 * time.Millisecond)

	t.Run("should attribute revenue to correct traffic sources", func(t *testing.T) {
		// Create 3 visitors with different traffic sources
		visitors := []struct {
			visitorID   string
			sessionID   string
			utmSource   string
			utmMedium   string
			utmCampaign string
			referrer    string
			revenue     int64 // in cents
		}{
			{
				visitorID:   "visitor-google-1",
				sessionID:   "session-google-1",
				utmSource:   "google",
				utmMedium:   "cpc",
				utmCampaign: "summer-sale",
				referrer:    "https://google.com/search",
				revenue:     9900, // $99.00
			},
			{
				visitorID:   "visitor-facebook-1",
				sessionID:   "session-facebook-1",
				utmSource:   "facebook",
				utmMedium:   "social",
				utmCampaign: "brand-awareness",
				referrer:    "https://facebook.com",
				revenue:     14900, // $149.00
			},
			{
				visitorID:   "visitor-direct-1",
				sessionID:   "session-direct-1",
				utmSource:   "",
				utmMedium:   "",
				utmCampaign: "",
				referrer:    "",
				revenue:     4900, // $49.00
			},
		}

		// Step 1: Send visitor events for each visitor
		for _, v := range visitors {
			builder := fixtures.NewEventBuilder().
				WithVisitorID(v.visitorID).
				WithSessionID(v.sessionID).
				WithPageURL("/").
				WithHost(project.Domain)

			if v.utmSource != "" {
				builder = builder.
					WithUTMSource(v.utmSource).
					WithUTMMedium(v.utmMedium).
					WithUTMCampaign(v.utmCampaign)
			}

			if v.referrer != "" {
				builder = builder.WithReferrer(v.referrer)
			}

			event := builder.Build()
			err := fixtures.SendEventToTestServer(t, tc, project, event)
			require.NoError(t, err, "Failed to send visitor event for %s", v.visitorID)
		}

		// Wait for events to be stored
		_, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 3, 10*time.Second)
		require.NoError(t, err, "Events should appear in ClickHouse")

		// Give time for attribution materialized views to update
		time.Sleep(2 * time.Second)

		// Step 2: Send payment events for each visitor
		for _, v := range visitors {
			payment := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
				WithVisitorID(v.visitorID).
				WithAmount(v.revenue).
				WithProductName("Test Product").
				Build()

			err := fixtures.SendPaymentToNATS(t, tc, payment)
			require.NoError(t, err, "Failed to send payment for %s", v.visitorID)
		}

		// Wait for payments to be processed
		succeededStatus := "succeeded"
		payments, err := fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			PaymentStatus: &succeededStatus,
		}, 3, 10*time.Second)
		require.NoError(t, err, "Payments should appear in ClickHouse")
		assert.Len(t, payments, 3, "Should have stored all 3 payments")

		// Give time for revenue materialized views to update
		time.Sleep(2 * time.Second)

		// Step 3: Verify revenue attribution by UTM source
		t.Run("revenue attributed to UTM source", func(t *testing.T) {
			dataPoints, err := revenueDataService.GetAttributionByUTM(ctx, project.ID, revenueTypes.TimeRangeLast7Days, "source")
			require.NoError(t, err, "Should get attribution by UTM source")
			require.NotEmpty(t, dataPoints, "Should have attribution data")

			// Find Google and Facebook in the results
			var googleRevenue, facebookRevenue *revenueTypes.UTMAttributionDataPoint
			for i := range dataPoints {
				switch dataPoints[i].UTMValue {
				case "google":
					googleRevenue = &dataPoints[i]
				case "facebook":
					facebookRevenue = &dataPoints[i]
				}
			}

			require.NotNil(t, googleRevenue, "Should have Google attribution")
			assert.Equal(t, int64(9900), googleRevenue.TotalRevenue, "Google revenue should be $99")
			assert.Equal(t, uint64(1), googleRevenue.PayingCustomers, "Google should have 1 paying customer")

			require.NotNil(t, facebookRevenue, "Should have Facebook attribution")
			assert.Equal(t, int64(14900), facebookRevenue.TotalRevenue, "Facebook revenue should be $149")
			assert.Equal(t, uint64(1), facebookRevenue.PayingCustomers, "Facebook should have 1 paying customer")

			t.Logf("✓ Google: $%.2f, Facebook: $%.2f",
				float64(googleRevenue.TotalRevenue)/100,
				float64(facebookRevenue.TotalRevenue)/100)
		})

		// Step 4: Verify revenue attribution by UTM medium
		t.Run("revenue attributed to UTM medium", func(t *testing.T) {
			dataPoints, err := revenueDataService.GetAttributionByUTM(ctx, project.ID, revenueTypes.TimeRangeLast7Days, "medium")
			require.NoError(t, err, "Should get attribution by UTM medium")
			require.NotEmpty(t, dataPoints, "Should have attribution data")

			// Find CPC and Social in the results
			var cpcRevenue, socialRevenue *revenueTypes.UTMAttributionDataPoint
			for i := range dataPoints {
				switch dataPoints[i].UTMValue {
				case "cpc":
					cpcRevenue = &dataPoints[i]
				case "social":
					socialRevenue = &dataPoints[i]
				}
			}

			require.NotNil(t, cpcRevenue, "Should have CPC attribution")
			assert.Equal(t, int64(9900), cpcRevenue.TotalRevenue, "CPC revenue should be $99")

			require.NotNil(t, socialRevenue, "Should have Social attribution")
			assert.Equal(t, int64(14900), socialRevenue.TotalRevenue, "Social revenue should be $149")

			t.Logf("✓ CPC: $%.2f, Social: $%.2f",
				float64(cpcRevenue.TotalRevenue)/100,
				float64(socialRevenue.TotalRevenue)/100)
		})

		// Step 5: Verify revenue attribution by traffic origin
		t.Run("revenue attributed to traffic origin", func(t *testing.T) {
			dataPoints, err := revenueDataService.GetAttributionByOrigin(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get attribution by origin")
			require.NotEmpty(t, dataPoints, "Should have attribution data")

			// Verify we have data for google.com and facebook.com
			var googleOrigin, facebookOrigin, directOrigin *revenueTypes.OriginAttributionDataPoint
			for i := range dataPoints {
				switch dataPoints[i].Origin {
				case "google.com":
					googleOrigin = &dataPoints[i]
				case "facebook.com":
					facebookOrigin = &dataPoints[i]
				case "Direct":
					directOrigin = &dataPoints[i]
				}
			}

			require.NotNil(t, googleOrigin, "Should have Google origin attribution")
			assert.Equal(t, int64(9900), googleOrigin.TotalRevenue, "Google origin revenue should be $99")

			require.NotNil(t, facebookOrigin, "Should have Facebook origin attribution")
			assert.Equal(t, int64(14900), facebookOrigin.TotalRevenue, "Facebook origin revenue should be $149")

			require.NotNil(t, directOrigin, "Should have Direct origin attribution")
			assert.Equal(t, int64(4900), directOrigin.TotalRevenue, "Direct origin revenue should be $49")

			t.Logf("✓ google.com: $%.2f, facebook.com: $%.2f, Direct: $%.2f",
				float64(googleOrigin.TotalRevenue)/100,
				float64(facebookOrigin.TotalRevenue)/100,
				float64(directOrigin.TotalRevenue)/100)
		})

		// Step 6: Verify dashboard metrics
		t.Run("dashboard shows correct revenue metrics", func(t *testing.T) {
			dashboard, err := revenueDataService.GetDashboardMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get dashboard metrics")
			require.NotNil(t, dashboard)

			// Total revenue should be sum of all payments: $99 + $149 + $49 = $297
			expectedTotalRevenue := int64(9900 + 14900 + 4900)
			assert.Equal(t, expectedTotalRevenue, dashboard.TotalRevenue, "Total revenue should be $297")
			assert.Equal(t, uint64(3), dashboard.TotalPayments, "Should have 3 payments")
			assert.Equal(t, uint64(3), dashboard.PayingCustomers, "Should have 3 paying customers")

			// Average order value: $297 / 3 = $99
			expectedAvgOrderValue := float64(expectedTotalRevenue) / 3.0
			assert.InDelta(t, expectedAvgOrderValue, dashboard.AvgOrderValue, 1.0, "Average order value should be ~$99")

			t.Logf("✓ Total Revenue: $%.2f, Total Payments: %d, Paying Customers: %d",
				float64(dashboard.TotalRevenue)/100,
				dashboard.TotalPayments,
				dashboard.PayingCustomers)
		})
	})

	t.Run("should handle first-touch attribution correctly", func(t *testing.T) {
		visitorID := "visitor-multi-touch"
		sessionID := "session-multi-touch-1"

		// First visit: comes from Google with UTM parameters
		firstVisit := fixtures.NewEventBuilder().
			WithVisitorID(visitorID).
			WithSessionID(sessionID).
			WithPageURL("/").
			WithHost(project.Domain).
			WithUTMSource("google").
			WithUTMMedium("cpc").
			WithUTMCampaign("first-campaign").
			WithReferrer("https://google.com/search").
			Build()

		err := fixtures.SendEventToTestServer(t, tc, project, firstVisit)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		// Second visit: comes from Facebook (different source)
		secondVisit := fixtures.NewEventBuilder().
			WithVisitorID(visitorID).
			WithSessionID(sessionID).
			WithPageURL("/products").
			WithHost(project.Domain).
			WithUTMSource("facebook").
			WithUTMMedium("social").
			WithUTMCampaign("second-campaign").
			WithReferrer("https://facebook.com").
			Build()

		err = fixtures.SendEventToTestServer(t, tc, project, secondVisit)
		require.NoError(t, err)

		// Wait for events
		_, err = fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
			VisitorID: &visitorID,
		}, 2, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// Send payment for this visitor
		payment := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(visitorID).
			WithAmount(19900). // $199
			WithProductName("Premium Product").
			Build()

		err = fixtures.SendPaymentToNATS(t, tc, payment)
		require.NoError(t, err)

		// Wait for payment
		succeededStatus := "succeeded"
		_, err = fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			VisitorID:     &visitorID,
			PaymentStatus: &succeededStatus,
		}, 1, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// Verify attribution goes to Google (first touch), not Facebook
		dataPoints, err := revenueDataService.GetAttributionByUTM(ctx, project.ID, revenueTypes.TimeRangeLast7Days, "source")
		require.NoError(t, err)

		// Find Google in the results
		var googleFound bool
		var googleRevenue int64
		for _, dp := range dataPoints {
			if dp.UTMValue == "google" {
				googleFound = true
				googleRevenue = dp.TotalRevenue
				break
			}
		}

		require.True(t, googleFound, "Should attribute revenue to Google (first touch)")
		assert.GreaterOrEqual(t, googleRevenue, int64(19900), "Google should have at least $199 in revenue")

		t.Logf("✓ First-touch attribution working: Revenue attributed to Google (first source), not Facebook (later source)")
	})

	t.Run("should track customer profile correctly", func(t *testing.T) {
		visitorID := "visitor-profile-test"
		sessionID := "session-profile-test"

		// Send visitor event
		event := fixtures.NewEventBuilder().
			WithVisitorID(visitorID).
			WithSessionID(sessionID).
			WithPageURL("/").
			WithHost(project.Domain).
			WithUTMSource("twitter").
			WithUTMMedium("social").
			WithUTMCampaign("profile-test").
			WithReferrer("https://twitter.com").
			Build()

		err := fixtures.SendEventToTestServer(t, tc, project, event)
		require.NoError(t, err)

		// Wait for event
		_, err = fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
			VisitorID: &visitorID,
		}, 1, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// Send two payments for this visitor
		payment1 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(visitorID).
			WithAmount(5000). // $50
			WithProductName("Product A").
			Build()

		payment2 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(visitorID).
			WithAmount(7500). // $75
			WithProductName("Product B").
			Build()

		err = fixtures.SendPaymentToNATS(t, tc, payment1)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		err = fixtures.SendPaymentToNATS(t, tc, payment2)
		require.NoError(t, err)

		// Wait for payments
		succeededStatus := "succeeded"
		_, err = fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			VisitorID:     &visitorID,
			PaymentStatus: &succeededStatus,
		}, 2, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// Get customer profile
		profile, err := revenueDataService.GetCustomerProfile(ctx, project.ID, visitorID)
		require.NoError(t, err)
		require.NotNil(t, profile)

		// Verify profile data
		assert.Equal(t, visitorID, profile.VisitorID)
		assert.Equal(t, int64(12500), profile.TotalRevenue, "Total revenue should be $125")
		assert.Equal(t, uint64(2), profile.PaymentCount, "Should have 2 payments")
		assert.InDelta(t, 6250.0, float64(profile.AvgOrderValue), 1.0, "Average order value should be $62.50")

		// Verify attribution
		require.NotNil(t, profile.FirstUTMSource)
		assert.Equal(t, "twitter", *profile.FirstUTMSource, "First UTM source should be twitter")

		t.Logf("✓ Customer Profile: Visitor %s, Revenue $%.2f, Payments: %d, First Source: %s",
			profile.VisitorID,
			float64(profile.TotalRevenue)/100,
			profile.PaymentCount,
			*profile.FirstUTMSource)
	})
}
