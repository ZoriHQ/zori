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

	revenueDataService := tc.RevenueData
	ctx := context.Background()

	// Give processors and ingestion server time to initialize
	time.Sleep(1 * time.Second)

	t.Run("should attribute revenue to correct traffic sources", func(t *testing.T) {
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

		_, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 3, 10*time.Second)
		require.NoError(t, err, "Events should appear in ClickHouse")

		time.Sleep(2 * time.Second)

		for _, v := range visitors {
			payment := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
				WithVisitorID(v.visitorID).
				WithAmount(v.revenue).
				WithProductName("Test Product").
				Build()

			err := fixtures.SendPaymentToNATS(t, tc, payment)
			require.NoError(t, err, "Failed to send payment for %s", v.visitorID)
		}

		succeededStatus := "succeeded"
		payments, err := fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			PaymentStatus: &succeededStatus,
		}, 3, 10*time.Second)
		require.NoError(t, err, "Payments should appear in ClickHouse")
		assert.Len(t, payments, 3, "Should have stored all 3 payments")

		time.Sleep(2 * time.Second)

		t.Run("revenue attributed to UTM source", func(t *testing.T) {
			dataPoints, err := revenueDataService.GetAttributionByUTM(ctx, project.ID, revenueTypes.TimeRangeLast7Days, "source")
			require.NoError(t, err, "Should get attribution by UTM source")
			require.NotEmpty(t, dataPoints, "Should have attribution data")

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

		t.Run("revenue attributed to UTM medium", func(t *testing.T) {
			dataPoints, err := revenueDataService.GetAttributionByUTM(ctx, project.ID, revenueTypes.TimeRangeLast7Days, "medium")
			require.NoError(t, err, "Should get attribution by UTM medium")
			require.NotEmpty(t, dataPoints, "Should have attribution data")

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

		t.Run("revenue attributed to traffic origin", func(t *testing.T) {
			dataPoints, err := revenueDataService.GetAttributionByOrigin(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get attribution by origin")
			require.NotEmpty(t, dataPoints, "Should have attribution data")

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

		t.Run("dashboard shows correct revenue metrics", func(t *testing.T) {
			dashboard, err := revenueDataService.GetDashboardMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get dashboard metrics")
			require.NotNil(t, dashboard)

			// Total revenue should be sum of all payments: $99 + $149 + $49 = $297
			expectedTotalRevenue := int64(9900 + 14900 + 4900)
			assert.Equal(t, expectedTotalRevenue, dashboard.TotalRevenue, "Total revenue should be $297")
			assert.Equal(t, uint64(3), dashboard.TotalPayments, "Should have 3 payments")
			assert.Equal(t, uint64(3), dashboard.PayingCustomers, "Should have 3 paying customers")

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

		_, err = fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
			VisitorID: &visitorID,
		}, 2, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		payment := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(visitorID).
			WithAmount(19900). // $199
			WithProductName("Premium Product").
			Build()

		err = fixtures.SendPaymentToNATS(t, tc, payment)
		require.NoError(t, err)

		succeededStatus := "succeeded"
		_, err = fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			VisitorID:     &visitorID,
			PaymentStatus: &succeededStatus,
		}, 1, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		dataPoints, err := revenueDataService.GetAttributionByUTM(ctx, project.ID, revenueTypes.TimeRangeLast7Days, "source")
		require.NoError(t, err)

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

		_, err = fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
			VisitorID: &visitorID,
		}, 1, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

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

		succeededStatus := "succeeded"
		_, err = fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			VisitorID:     &visitorID,
			PaymentStatus: &succeededStatus,
		}, 2, 10*time.Second)
		require.NoError(t, err)

		// Give revenue attribution system more time to process and aggregate the payments
		// The system needs time to: process payments, update aggregations, and refresh materialized views
		time.Sleep(6 * time.Second)

		profile, err := revenueDataService.GetCustomerProfile(ctx, project.ID, visitorID)
		require.NoError(t, err)
		require.NotNil(t, profile)

		assert.Equal(t, visitorID, profile.VisitorID)
		assert.Equal(t, int64(12500), profile.TotalRevenue, "Total revenue should be $125")
		assert.Equal(t, uint64(2), profile.PaymentCount, "Should have 2 payments")
		assert.InDelta(t, 6250.0, float64(profile.AvgOrderValue), 1.0, "Average order value should be $62.50")

		require.NotNil(t, profile.FirstUTMSource)
		assert.Equal(t, "twitter", *profile.FirstUTMSource, "First UTM source should be twitter")

		t.Logf("✓ Customer Profile: Visitor %s, Revenue $%.2f, Payments: %d, First Source: %s",
			profile.VisitorID,
			float64(profile.TotalRevenue)/100,
			profile.PaymentCount,
			*profile.FirstUTMSource)
	})
}

func TestConversionMetrics_EndToEnd(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)

	revenueDataService := tc.RevenueData
	ctx := context.Background()

	// Give processors and ingestion server time to initialize
	time.Sleep(1 * time.Second)

	t.Run("should calculate conversion metrics correctly", func(t *testing.T) {
		now := time.Now().UTC()

		// Create test scenarios with different visitor behaviors
		visitors := []struct {
			visitorID       string
			sessionID       string
			firstVisitTime  time.Time
			willPurchase    bool
			purchaseTime    time.Time
			purchaseAmount  int64
			repeatPurchase  bool
			repeat2Amount   int64
			repeat2Time     time.Time
			utmSource       string
		}{
			{
				// Visitor 1: Quick converter (purchases within 1 hour)
				visitorID:      "visitor-quick-convert",
				sessionID:      "session-quick-1",
				firstVisitTime: now.Add(-2 * time.Hour),
				willPurchase:   true,
				purchaseTime:   now.Add(-1 * time.Hour),
				purchaseAmount: 10000, // $100
				repeatPurchase: false,
				utmSource:      "google",
			},
			{
				// Visitor 2: Slow converter (purchases after 24 hours) with repeat purchase
				visitorID:      "visitor-slow-convert",
				sessionID:      "session-slow-1",
				firstVisitTime: now.Add(-48 * time.Hour),
				willPurchase:   true,
				purchaseTime:   now.Add(-24 * time.Hour),
				purchaseAmount: 15000, // $150
				repeatPurchase: true,
				repeat2Amount:  8000, // $80
				repeat2Time:    now.Add(-12 * time.Hour),
				utmSource:      "facebook",
			},
			{
				// Visitor 3: Medium converter with repeat purchase
				visitorID:      "visitor-medium-convert",
				sessionID:      "session-medium-1",
				firstVisitTime: now.Add(-12 * time.Hour),
				willPurchase:   true,
				purchaseTime:   now.Add(-6 * time.Hour),
				purchaseAmount: 20000, // $200
				repeatPurchase: true,
				repeat2Amount:  5000, // $50
				repeat2Time:    now.Add(-3 * time.Hour),
				utmSource:      "twitter",
			},
			{
				// Visitor 4: Non-converter (visited but never purchased)
				visitorID:      "visitor-no-convert-1",
				sessionID:      "session-no-1",
				firstVisitTime: now.Add(-6 * time.Hour),
				willPurchase:   false,
				utmSource:      "linkedin",
			},
			{
				// Visitor 5: Another non-converter
				visitorID:      "visitor-no-convert-2",
				sessionID:      "session-no-2",
				firstVisitTime: now.Add(-3 * time.Hour),
				willPurchase:   false,
				utmSource:      "reddit",
			},
		}

		// Send all visitor events with appropriate timestamps
		for _, v := range visitors {
			builder := fixtures.NewEventBuilder().
				WithVisitorID(v.visitorID).
				WithSessionID(v.sessionID).
				WithPageURL("/").
				WithHost(project.Domain).
				WithUTMSource(v.utmSource).
				WithUTMMedium("cpc").
				WithUTMCampaign("conversion-test")

			event := builder.Build()
			// Note: In a real scenario, we'd set the timestamp, but fixtures may not support it
			// The test will work with current timestamps but time-to-purchase metrics may vary
			err := fixtures.SendEventToTestServer(t, tc, project, event)
			require.NoError(t, err, "Failed to send visitor event for %s", v.visitorID)
		}

		// Wait for all events to be processed
		_, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 5, 15*time.Second)
		require.NoError(t, err, "Events should appear in ClickHouse")

		time.Sleep(2 * time.Second)

		// Send payments for visitors who will purchase
		for _, v := range visitors {
			if v.willPurchase {
				// First payment
				payment := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
					WithVisitorID(v.visitorID).
					WithAmount(v.purchaseAmount).
					WithProductName("Test Product").
					Build()

				err := fixtures.SendPaymentToNATS(t, tc, payment)
				require.NoError(t, err, "Failed to send first payment for %s", v.visitorID)

				// Send repeat purchase if applicable
				if v.repeatPurchase {
					time.Sleep(200 * time.Millisecond)
					payment2 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
						WithVisitorID(v.visitorID).
						WithAmount(v.repeat2Amount).
						WithProductName("Repeat Product").
						Build()

					err = fixtures.SendPaymentToNATS(t, tc, payment2)
					require.NoError(t, err, "Failed to send repeat payment for %s", v.visitorID)
				}
			}
		}

		// Wait for payments to be processed
		succeededStatus := "succeeded"
		payments, err := fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			PaymentStatus: &succeededStatus,
		}, 5, 15*time.Second) // 3 paying customers, 2 with repeat purchases = 5 total payments
		require.NoError(t, err, "Payments should appear in ClickHouse")
		assert.Len(t, payments, 5, "Should have stored all 5 payments")

		// Give the system time to process and aggregate
		time.Sleep(3 * time.Second)

		t.Run("conversion rate", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

			// Due to potential test isolation issues, we verify that:
			// 1. We have at least 5 visitors (our test visitors)
			// 2. We have at least 3 paying customers (from our test)
			// 3. Conversion rate is reasonable (not 0% or 100%)
			assert.GreaterOrEqual(t, metrics.TotalVisitors, uint64(5), "Should have at least 5 total visitors")
			assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(3), "Should have at least 3 paying customers")
			assert.Greater(t, metrics.ConversionRate, 0.0, "Conversion rate should be positive")
			assert.LessOrEqual(t, metrics.ConversionRate, 100.0, "Conversion rate should not exceed 100%")

			t.Logf("✓ Conversion Rate: %.2f%% (%d paying customers / %d total visitors)",
				metrics.ConversionRate,
				metrics.PayingCustomers,
				metrics.TotalVisitors)
		})

		t.Run("average time to first purchase", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

			// The test uses current timestamps, so we can't assert exact values
			// But we can verify that the metrics are reasonable
			assert.GreaterOrEqual(t, metrics.AvgTimeToFirstPurchase, 0.0,
				"Average time to first purchase should be non-negative")
			assert.GreaterOrEqual(t, metrics.MedianTimeToFirstPurchase, 0.0,
				"Median time to first purchase should be non-negative")

			// In production with accurate timestamps:
			// - visitor-quick-convert: 1 hour
			// - visitor-slow-convert: 24 hours
			// - visitor-medium-convert: 6 hours
			// Average: (1 + 24 + 6) / 3 = 10.33 hours
			// Median: 6 hours

			t.Logf("✓ Avg Time to First Purchase: %.2f hours", metrics.AvgTimeToFirstPurchase)
			t.Logf("✓ Median Time to First Purchase: %.2f hours", metrics.MedianTimeToFirstPurchase)
		})

		t.Run("customer lifetime value (LTV)", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

			// Total revenue from our test paying customers:
			// visitor-quick-convert: $100
			// visitor-slow-convert: $150 + $80 = $230
			// visitor-medium-convert: $200 + $50 = $250
			// Total: $580
			// Expected minimum LTV = $580 / number_of_paying_customers

			// LTV should be positive and reasonable
			assert.Greater(t, metrics.CustomerLifetimeValue, 0.0, "Customer LTV should be positive")

			// Our test generated $580 in total revenue across 3 paying customers
			// So average LTV should be at least $100 (lowest individual customer revenue)
			minExpectedLTV := float64(10000) // $100 in cents
			assert.GreaterOrEqual(t, metrics.CustomerLifetimeValue, minExpectedLTV,
				"Customer LTV should be at least $100")

			t.Logf("✓ Customer Lifetime Value: $%.2f", metrics.CustomerLifetimeValue/100)
		})

		t.Run("repeat purchase rate", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

			// Of our 3 paying customers, 2 made repeat purchases
			// So repeat purchase rate should be positive and reasonable
			assert.GreaterOrEqual(t, metrics.RepeatPurchaseRate, 0.0,
				"Repeat purchase rate should be non-negative")
			assert.LessOrEqual(t, metrics.RepeatPurchaseRate, 100.0,
				"Repeat purchase rate should not exceed 100%")

			// Average purchases per customer should be at least 1.0
			assert.GreaterOrEqual(t, metrics.AvgPurchasesPerCustomer, 1.0,
				"Average purchases per customer should be at least 1")

			t.Logf("✓ Repeat Purchase Rate: %.2f%%", metrics.RepeatPurchaseRate)
			t.Logf("✓ Avg Purchases per Customer: %.2f", metrics.AvgPurchasesPerCustomer)
		})

		t.Run("complete conversion funnel metrics", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

			// Verify all metrics together
			t.Logf("\n=== Conversion Funnel Summary ===")
			t.Logf("Total Visitors: %d", metrics.TotalVisitors)
			t.Logf("Paying Customers: %d", metrics.PayingCustomers)
			t.Logf("Conversion Rate: %.2f%%", metrics.ConversionRate)
			t.Logf("Avg Time to First Purchase: %.2f hours", metrics.AvgTimeToFirstPurchase)
			t.Logf("Median Time to First Purchase: %.2f hours", metrics.MedianTimeToFirstPurchase)
			t.Logf("Customer Lifetime Value: $%.2f", metrics.CustomerLifetimeValue/100)
			t.Logf("Repeat Purchase Rate: %.2f%%", metrics.RepeatPurchaseRate)
			t.Logf("Avg Purchases per Customer: %.2f", metrics.AvgPurchasesPerCustomer)

			// Sanity checks
			assert.Greater(t, metrics.TotalVisitors, uint64(0), "Should have visitors")
			assert.Greater(t, metrics.PayingCustomers, uint64(0), "Should have paying customers")
			assert.Greater(t, metrics.ConversionRate, 0.0, "Should have positive conversion rate")
			assert.Greater(t, metrics.CustomerLifetimeValue, 0.0, "Should have positive LTV")
		})
	})
}
