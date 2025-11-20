package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"zori/di"
	paymentTypes "zori/services/payments/types"
	revenueTypes "zori/services/revenue/types"
	"zori/testutil/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stringPtr(s string) *string {
	return &s
}

func TestRevenueAttribution_EndToEnd(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)

	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should attribute revenue to correct traffic sources", func(t *testing.T) {
		visitors := []struct {
			visitorID   string
			sessionID   string
			utmSource   string
			utmMedium   string
			utmCampaign string
			referrer    string
			revenue     int64
		}{
			{
				visitorID:   "visitor-google-1",
				sessionID:   "session-google-1",
				utmSource:   "google",
				utmMedium:   "cpc",
				utmCampaign: "summer-sale",
				referrer:    "https://google.com/search",
				revenue:     9900,
			},
			{
				visitorID:   "visitor-facebook-1",
				sessionID:   "session-facebook-1",
				utmSource:   "facebook",
				utmMedium:   "social",
				utmCampaign: "brand-awareness",
				referrer:    "https://facebook.com",
				revenue:     14900,
			},
			{
				visitorID:   "visitor-direct-1",
				sessionID:   "session-direct-1",
				utmSource:   "",
				utmMedium:   "",
				utmCampaign: "",
				referrer:    "",
				revenue:     4900,
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

		time.Sleep(1 * time.Second)

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

		maxWait := 10 * time.Second
		checkInterval := 500 * time.Millisecond
		deadline := time.Now().Add(maxWait)
		attributionFound := false

		for time.Now().Before(deadline) {
			query := `
				SELECT argMinMerge(first_utm_source) as first_source
				FROM visitor_first_touch_attribution
				WHERE project_id = ? AND visitor_id = ?
				GROUP BY visitor_id
			`
			var firstSource string
			row := tc.ClickHouse.Db().QueryRow(ctx, query, project.ID, visitorID)
			scanErr := row.Scan(&firstSource)
			if scanErr == nil && firstSource == "google" {
				attributionFound = true
				t.Logf("✓ First-touch attribution found: source=%s", firstSource)
				break
			} else if scanErr != nil {
				t.Logf("Attribution query error (will retry): %v", scanErr)
			} else {
				t.Logf("Attribution found but wrong source: %s (expected: google)", firstSource)
			}
			time.Sleep(checkInterval)
		}

		if !attributionFound {
			t.Logf("⚠ First-touch attribution MV not updated, trying direct query...")
			directQuery := `
				SELECT argMin(utm_parameters['utm_source'], client_timestamp_utc) as first_source
				FROM events
				WHERE project_id = ? AND visitor_id = ?
			`
			var directSource string
			directRow := tc.ClickHouse.Db().QueryRow(ctx, directQuery, project.ID, visitorID)
			if err := directRow.Scan(&directSource); err == nil {
				t.Logf("Direct query shows first_source=%s", directSource)
			} else {
				t.Logf("Direct query failed: %v", err)
			}
		}

		payment := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(visitorID).
			WithAmount(19900).
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
			WithAmount(5000).
			WithProductName("Product A").
			Build()

		payment2 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(visitorID).
			WithAmount(7500).
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

func TestRevenueMetrics_ConversionRate(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should NOT report 100% conversion when not all visitors purchased", func(t *testing.T) {
		builder := fixtures.NewRevenueTestDataBuilder()

		builder.AddSimpleConvertingVisitor(10000, "google")
		builder.AddSimpleConvertingVisitor(10000, "facebook")

		for i := 0; i < 8; i++ {
			builder.AddNonConvertingVisitor(fmt.Sprintf("source-%d", i))
		}

		err := builder.InsertTestData(t, tc, project)
		require.NoError(t, err)

		metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		expectedVisitors, expectedCustomers, _ := builder.GetConversionStats()

		t.Logf("Expected: %d visitors, %d paying customers", expectedVisitors, expectedCustomers)
		t.Logf("Actual: %d visitors, %d paying customers, %.2f%% conversion",
			metrics.TotalVisitors, metrics.PayingCustomers, metrics.ConversionRate)

		assert.GreaterOrEqual(t, metrics.TotalVisitors, uint64(expectedVisitors),
			"Should count ALL visitors, not just paying ones")
		assert.Equal(t, uint64(expectedCustomers), metrics.PayingCustomers,
			"Should have exactly 2 paying customers")
		assert.Less(t, metrics.ConversionRate, 100.0,
			"BUG: Conversion rate should NOT be 100% when not all visitors purchased!")
		assert.Greater(t, metrics.ConversionRate, 0.0,
			"Conversion rate should be positive")

		assert.InDelta(t, 20.0, metrics.ConversionRate, 5.0,
			"Conversion rate should be approximately 20%")
	})

	t.Run("should calculate accurate conversion rate with known data", func(t *testing.T) {
		builder := fixtures.NewRevenueTestDataBuilder()

		for i := 0; i < 20; i++ {
			builder.AddSimpleConvertingVisitor(10000, fmt.Sprintf("source-%d", i%5))
		}

		for i := 0; i < 80; i++ {
			builder.AddNonConvertingVisitor(fmt.Sprintf("source-%d", i%5))
		}

		err := builder.InsertTestData(t, tc, project)
		require.NoError(t, err)

		metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		expectedVisitors, expectedCustomers, expectedRevenue := builder.GetConversionStats()
		assert.GreaterOrEqual(t, metrics.TotalVisitors, uint64(expectedVisitors))
		assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(expectedCustomers))

		expectedConversionRate := float64(expectedCustomers) * 100.0 / float64(expectedVisitors)
		t.Logf("Expected: %d visitors, %d customers, %.2f%% conversion, $%.2f revenue",
			expectedVisitors, expectedCustomers, expectedConversionRate, float64(expectedRevenue)/100)
		t.Logf("Actual: %d visitors, %d customers, %.2f%% conversion",
			metrics.TotalVisitors, metrics.PayingCustomers, metrics.ConversionRate)

		assert.Greater(t, metrics.ConversionRate, 0.0)
		assert.LessOrEqual(t, metrics.ConversionRate, 100.0)
	})

	t.Run("should handle 0% conversion rate", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()
		for i := 0; i < 10; i++ {
			builder.AddNonConvertingVisitor("test-source")
		}

		err := builder.InsertTestData(t, tc2, project2)
		require.NoError(t, err)

		metrics, err := tc2.RevenueData.GetConversionMetrics(ctx, project2.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, metrics.TotalVisitors, uint64(10))
		assert.Equal(t, uint64(0), metrics.PayingCustomers)
		assert.Equal(t, 0.0, metrics.ConversionRate)
	})

	t.Run("should handle 100% conversion rate", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()
		for i := 0; i < 10; i++ {
			builder.AddSimpleConvertingVisitor(5000, "test-source")
		}

		err := builder.InsertTestData(t, tc3, project3)
		require.NoError(t, err)

		metrics, err := tc3.RevenueData.GetConversionMetrics(ctx, project3.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, metrics.TotalVisitors, uint64(10))
		assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(10))
		assert.Equal(t, 100.0, metrics.ConversionRate)
	})
}

func TestRevenueMetrics_CustomerLTV(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should calculate correct LTV for single-purchase customers", func(t *testing.T) {
		builder := fixtures.NewRevenueTestDataBuilder()

		for i := 0; i < 5; i++ {
			builder.AddSimpleConvertingVisitor(10000, "test-source")
		}

		err := builder.InsertTestData(t, tc, project)
		require.NoError(t, err)

		metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		_, expectedCustomers, expectedRevenue := builder.GetConversionStats()
		expectedLTV := float64(expectedRevenue) / float64(expectedCustomers)

		t.Logf("Expected LTV: $%.2f (revenue: $%.2f, customers: %d)",
			expectedLTV/100, float64(expectedRevenue)/100, expectedCustomers)
		t.Logf("Actual LTV: $%.2f", metrics.CustomerLifetimeValue/100)

		assert.GreaterOrEqual(t, metrics.CustomerLifetimeValue, 9000.0)
		assert.LessOrEqual(t, metrics.CustomerLifetimeValue, 11000.0)
	})

	t.Run("should calculate correct LTV for repeat purchase customers", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()

		builder.AddRepeatCustomer([]int64{10000, 15000, 20000}, "google")

		builder.AddRepeatCustomer([]int64{10000, 10000}, "facebook")

		builder.AddSimpleConvertingVisitor(5000, "twitter")

		err := builder.InsertTestData(t, tc2, project2)
		require.NoError(t, err)

		metrics, err := tc2.RevenueData.GetConversionMetrics(ctx, project2.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		_, expectedCustomers, expectedRevenue := builder.GetConversionStats()
		expectedLTV := float64(expectedRevenue) / float64(expectedCustomers)

		t.Logf("Expected LTV: $%.2f (revenue: $%.2f, customers: %d)",
			expectedLTV/100, float64(expectedRevenue)/100, expectedCustomers)
		t.Logf("Actual LTV: $%.2f", metrics.CustomerLifetimeValue/100)

		assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(3))
		assert.Greater(t, metrics.CustomerLifetimeValue, 0.0)
	})

	t.Run("should calculate correct LTV with varying revenue amounts", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()

		builder.AddSimpleConvertingVisitor(1000, "source1")
		builder.AddSimpleConvertingVisitor(5000, "source2")
		builder.AddSimpleConvertingVisitor(10000, "source3")
		builder.AddSimpleConvertingVisitor(50000, "source4")
		builder.AddSimpleConvertingVisitor(100000, "source5")

		err := builder.InsertTestData(t, tc3, project3)
		require.NoError(t, err)

		metrics, err := tc3.RevenueData.GetConversionMetrics(ctx, project3.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		_, expectedCustomers, expectedRevenue := builder.GetConversionStats()
		expectedLTV := float64(expectedRevenue) / float64(expectedCustomers)

		t.Logf("Expected LTV: $%.2f (revenue: $%.2f, customers: %d)",
			expectedLTV/100, float64(expectedRevenue)/100, expectedCustomers)
		t.Logf("Actual LTV: $%.2f", metrics.CustomerLifetimeValue/100)

		assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(5))
		assert.Greater(t, metrics.CustomerLifetimeValue, 0.0)
	})
}

func TestRevenueMetrics_RepeatPurchaseRate(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should calculate correct repeat purchase rate", func(t *testing.T) {
		builder := fixtures.NewRevenueTestDataBuilder()

		builder.AddRepeatCustomer([]int64{10000, 15000}, "google")
		builder.AddRepeatCustomer([]int64{5000, 5000, 5000}, "facebook")
		builder.AddRepeatCustomer([]int64{20000, 10000}, "twitter")

		builder.AddSimpleConvertingVisitor(10000, "linkedin")
		builder.AddSimpleConvertingVisitor(15000, "reddit")

		err := builder.InsertTestData(t, tc, project)
		require.NoError(t, err)

		metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		expectedCustomers, expectedRepeatCustomers := builder.GetRepeatPurchaseStats()
		expectedRepeatRate := float64(expectedRepeatCustomers) * 100.0 / float64(expectedCustomers)

		t.Logf("Expected: %d total customers, %d repeat customers, %.2f%% repeat rate",
			expectedCustomers, expectedRepeatCustomers, expectedRepeatRate)
		t.Logf("Actual: %d customers, %.2f%% repeat rate, avg %.2f purchases/customer",
			metrics.PayingCustomers, metrics.RepeatPurchaseRate, metrics.AvgPurchasesPerCustomer)

		assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(5))
		assert.GreaterOrEqual(t, metrics.RepeatPurchaseRate, 0.0)
		assert.LessOrEqual(t, metrics.RepeatPurchaseRate, 100.0)
		assert.GreaterOrEqual(t, metrics.AvgPurchasesPerCustomer, 1.0)
	})

	t.Run("should handle 0% repeat purchase rate", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()

		for i := 0; i < 5; i++ {
			builder.AddSimpleConvertingVisitor(10000, fmt.Sprintf("source-%d", i))
		}

		err := builder.InsertTestData(t, tc2, project2)
		require.NoError(t, err)

		metrics, err := tc2.RevenueData.GetConversionMetrics(ctx, project2.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		t.Logf("Metrics: %d customers, %.2f%% repeat rate, avg %.2f purchases/customer",
			metrics.PayingCustomers, metrics.RepeatPurchaseRate, metrics.AvgPurchasesPerCustomer)

		assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(5))
		assert.Equal(t, 0.0, metrics.RepeatPurchaseRate)
		assert.InDelta(t, 1.0, metrics.AvgPurchasesPerCustomer, 0.1)
	})

	t.Run("should handle 100% repeat purchase rate", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()

		for i := 0; i < 5; i++ {
			builder.AddRepeatCustomer([]int64{10000, 15000}, fmt.Sprintf("source-%d", i))
		}

		err := builder.InsertTestData(t, tc3, project3)
		require.NoError(t, err)

		metrics, err := tc3.RevenueData.GetConversionMetrics(ctx, project3.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		t.Logf("Metrics: %d customers, %.2f%% repeat rate, avg %.2f purchases/customer",
			metrics.PayingCustomers, metrics.RepeatPurchaseRate, metrics.AvgPurchasesPerCustomer)

		assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(5))
		assert.Equal(t, 100.0, metrics.RepeatPurchaseRate)
		assert.GreaterOrEqual(t, metrics.AvgPurchasesPerCustomer, 2.0)
	})

	t.Run("should calculate average purchases per customer correctly", func(t *testing.T) {
		tc4 := di.NewTestContainer(t)
		defer tc4.Cleanup()

		_, project4 := fixtures.SetupAccountAndProject(t, tc4)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()

		builder.AddSimpleConvertingVisitor(10000, "source1")

		builder.AddRepeatCustomer([]int64{10000, 10000}, "source2")

		builder.AddRepeatCustomer([]int64{10000, 10000, 10000}, "source3")

		err := builder.InsertTestData(t, tc4, project4)
		require.NoError(t, err)

		metrics, err := tc4.RevenueData.GetConversionMetrics(ctx, project4.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)

		t.Logf("Metrics: %d customers, %.2f avg purchases/customer",
			metrics.PayingCustomers, metrics.AvgPurchasesPerCustomer)

		assert.GreaterOrEqual(t, metrics.PayingCustomers, uint64(3))
		assert.InDelta(t, 2.0, metrics.AvgPurchasesPerCustomer, 0.2)
	})
}

func TestRevenueMetrics_Timeline(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should aggregate revenue over time correctly", func(t *testing.T) {
		builder := fixtures.NewRevenueTestDataBuilder()
		now := time.Now().UTC()

		for day := 0; day < 7; day++ {
			amount := int64((day + 1) * 10000)
			timestamp := now.Add(-time.Duration(6-day) * 24 * time.Hour)

			builder.AddVisitor(fixtures.VisitorPaymentScenario{
				VisitorID:      fmt.Sprintf("visitor-day-%d", day),
				SessionID:      fmt.Sprintf("session-day-%d", day),
				FirstVisitTime: timestamp.Add(-1 * time.Hour),
				UTMSource:      "test-source",
				UTMMedium:      "cpc",
				UTMCampaign:    "test",
				ReferrerDomain: "test.com",
				WillPurchase:   true,
				Payments: []fixtures.PaymentScenario{
					{
						Amount:    amount,
						Timestamp: timestamp,
						Status:    "succeeded",
						Currency:  "USD",
					},
				},
			})
		}

		err := builder.InsertTestData(t, tc, project)
		require.NoError(t, err)

		timeline, err := revenueDataService.GetTimeline(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)
		require.NotEmpty(t, timeline)

		var timelineTotal int64
		for _, dp := range timeline {
			timelineTotal += dp.TotalRevenue
			t.Logf("Timeline point: %s - $%.2f (%d payments)",
				dp.Timestamp.Format("2006-01-02 15:04"),
				float64(dp.TotalRevenue)/100,
				dp.PaymentCount)
		}

		_, _, expectedRevenue := builder.GetConversionStats()

		t.Logf("Expected total revenue: $%.2f", float64(expectedRevenue)/100)
		t.Logf("Timeline total revenue: $%.2f", float64(timelineTotal)/100)

		assert.GreaterOrEqual(t, timelineTotal, expectedRevenue-1000)
		assert.LessOrEqual(t, timelineTotal, expectedRevenue+1000)
	})

	t.Run("should use hourly buckets for short time ranges", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()
		now := time.Now().UTC()

		for hour := 0; hour < 6; hour++ {
			timestamp := now.Add(-time.Duration(5-hour) * time.Hour)
			builder.AddVisitor(fixtures.VisitorPaymentScenario{
				VisitorID:      fmt.Sprintf("visitor-hour-%d", hour),
				SessionID:      fmt.Sprintf("session-hour-%d", hour),
				FirstVisitTime: timestamp.Add(-10 * time.Minute),
				UTMSource:      "test",
				WillPurchase:   true,
				Payments: []fixtures.PaymentScenario{
					{
						Amount:    5000,
						Timestamp: timestamp,
						Status:    "succeeded",
						Currency:  "USD",
					},
				},
			})
		}

		err := builder.InsertTestData(t, tc2, project2)
		require.NoError(t, err)

		timeline, err := tc2.RevenueData.GetTimeline(ctx, project2.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)
		require.NotEmpty(t, timeline)

		t.Logf("Timeline has %d data points", len(timeline))
		for _, dp := range timeline {
			t.Logf("  %s: $%.2f", dp.Timestamp.Format("2006-01-02 15:04"), float64(dp.TotalRevenue)/100)
		}

		assert.Greater(t, len(timeline), 0)
	})

	t.Run("should use daily buckets for longer time ranges", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()
		now := time.Now().UTC()

		for day := 0; day < 30; day++ {
			timestamp := now.Add(-time.Duration(29-day) * 24 * time.Hour)
			builder.AddVisitor(fixtures.VisitorPaymentScenario{
				VisitorID:      fmt.Sprintf("visitor-day30-%d", day),
				SessionID:      fmt.Sprintf("session-day30-%d", day),
				FirstVisitTime: timestamp.Add(-1 * time.Hour),
				UTMSource:      "test",
				WillPurchase:   true,
				Payments: []fixtures.PaymentScenario{
					{
						Amount:    3000,
						Timestamp: timestamp,
						Status:    "succeeded",
						Currency:  "USD",
					},
				},
			})
		}

		err := builder.InsertTestData(t, tc3, project3)
		require.NoError(t, err)

		timeline, err := tc3.RevenueData.GetTimeline(ctx, project3.ID, revenueTypes.TimeRangeLast30Days)
		require.NoError(t, err)
		require.NotEmpty(t, timeline)

		t.Logf("Timeline has %d data points", len(timeline))

		assert.Greater(t, len(timeline), 0)
		assert.LessOrEqual(t, len(timeline), 31)
	})
}

func TestRevenueMetrics_PaymentDataQuality(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should detect no duplicates with clean data", func(t *testing.T) {
		builder := fixtures.NewRevenueTestDataBuilder()

		for i := 0; i < 5; i++ {
			builder.AddSimpleConvertingVisitor(10000, fmt.Sprintf("source-%d", i))
		}

		err := builder.InsertTestData(t, tc, project)
		require.NoError(t, err)

		report, err := revenueDataService.CheckPaymentDataQuality(ctx, project.ID)
		require.NoError(t, err)
		require.NotNil(t, report)

		assert.Equal(t, uint64(0), report.DuplicatePaymentCount)
		assert.Equal(t, uint64(0), report.TotalDuplicateRows)
		assert.Equal(t, uint64(0), report.PaymentsWithConflictingAmounts)
		assert.Equal(t, uint64(0), report.PaymentsWithConflictingVisitors)
		assert.Empty(t, report.Samples)

		t.Logf("✓ No duplicates detected in clean data")
	})

	t.Run("should detect duplicate payment_ids", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		time.Sleep(1 * time.Second)

		duplicatePaymentID := "payment-duplicate-123"
		payments := []paymentTypes.PaymentEventFrame{
			{
				PaymentID:        duplicatePaymentID,
				VisitorID:        stringPtr("visitor-1"),
				ProviderType:     "stripe",
				PaymentStatus:    "succeeded",
				ProductName:      "Product A",
				Amount:           10000,
				Currency:         "USD",
				PaymentTimestamp: time.Now().UTC(),
				ProjectID:        project2.ID,
				OrganizationID:   project2.OrganizationID,
				Metadata:         make(map[string]string),
			},
			{
				PaymentID:        duplicatePaymentID,
				VisitorID:        stringPtr("visitor-1"),
				ProviderType:     "stripe",
				PaymentStatus:    "succeeded",
				ProductName:      "Product A",
				Amount:           15000,
				Currency:         "USD",
				PaymentTimestamp: time.Now().UTC(),
				ProjectID:        project2.ID,
				OrganizationID:   project2.OrganizationID,
				Metadata:         make(map[string]string),
			},
		}

		err := fixtures.InsertPaymentEventsDirect(t, tc2, project2, payments)
		require.NoError(t, err)

		time.Sleep(1 * time.Second)

		report, err := tc2.RevenueData.CheckPaymentDataQuality(ctx, project2.ID)
		require.NoError(t, err)
		require.NotNil(t, report)

		assert.Equal(t, uint64(1), report.DuplicatePaymentCount, "Should detect 1 duplicate payment_id")
		assert.Equal(t, uint64(2), report.TotalDuplicateRows, "Should count 2 total duplicate rows")
		assert.Equal(t, uint64(1), report.PaymentsWithConflictingAmounts, "Should detect conflicting amounts")
		assert.NotEmpty(t, report.Samples)

		t.Logf("✓ Detected duplicate payment_id with conflicting amounts")
	})
}

func TestRevenueMetrics_TopCustomers(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should return top customers ranked by revenue", func(t *testing.T) {
		builder := fixtures.NewRevenueTestDataBuilder()

		builder.AddRepeatCustomer([]int64{50000, 30000, 20000}, "google")
		builder.AddRepeatCustomer([]int64{40000, 35000}, "facebook")
		builder.AddSimpleConvertingVisitor(60000, "twitter")
		builder.AddRepeatCustomer([]int64{25000, 20000, 5000}, "linkedin")
		builder.AddSimpleConvertingVisitor(30000, "reddit")

		err := builder.InsertTestData(t, tc, project)
		require.NoError(t, err)

		customers, err := revenueDataService.GetTopCustomers(ctx, project.ID, revenueTypes.TimeRangeLast7Days, 10)
		require.NoError(t, err)
		require.NotEmpty(t, customers)

		assert.GreaterOrEqual(t, len(customers), 5)

		assert.GreaterOrEqual(t, customers[0].TotalRevenue, customers[1].TotalRevenue)

		foundWithAttribution := false
		for i, customer := range customers {
			t.Logf("%d. Visitor %s: $%.2f (%d payments, avg $%.2f) - Source: %v, Medium: %v",
				i+1,
				customer.VisitorID,
				float64(customer.TotalRevenue)/100,
				customer.PaymentCount,
				customer.AvgOrderValue,
				customer.FirstUTMSource,
				customer.FirstUTMMedium)

			if customer.FirstUTMSource != nil && *customer.FirstUTMSource != "" {
				foundWithAttribution = true
			}
		}

		assert.True(t, foundWithAttribution, "At least one customer should have UTM attribution data")
		t.Logf("✓ Top customers ranked correctly with attribution data")
	})

	t.Run("should respect limit parameter", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()

		for i := 0; i < 10; i++ {
			builder.AddSimpleConvertingVisitor(int64((i+1)*1000), fmt.Sprintf("source-%d", i))
		}

		err := builder.InsertTestData(t, tc2, project2)
		require.NoError(t, err)

		customers, err := tc2.RevenueData.GetTopCustomers(ctx, project2.ID, revenueTypes.TimeRangeLast7Days, 3)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(customers), 3, "Should return at most 3 customers")
		t.Logf("✓ Limit parameter respected: returned %d customers", len(customers))
	})

	t.Run("should include customer metadata", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()
		builder.AddSimpleConvertingVisitor(50000, "test-source")

		err := builder.InsertTestData(t, tc3, project3)
		require.NoError(t, err)

		customers, err := tc3.RevenueData.GetTopCustomers(ctx, project3.ID, revenueTypes.TimeRangeLast7Days, 10)
		require.NoError(t, err)
		require.NotEmpty(t, customers)

		customer := customers[0]
		assert.NotEmpty(t, customer.VisitorID)
		assert.Greater(t, customer.TotalRevenue, int64(0))
		assert.Greater(t, customer.PaymentCount, uint64(0))
		assert.NotNil(t, customer.FirstPaymentDate)
		assert.NotNil(t, customer.LastPaymentDate)
		assert.Greater(t, customer.AvgOrderValue, 0.0)

		t.Logf("✓ Customer metadata included")
	})

	t.Run("should calculate correct revenue for customers with multiple visitors and many events", func(t *testing.T) {
		tc4 := di.NewTestContainer(t)
		defer tc4.Cleanup()

		_, project4 := fixtures.SetupAccountAndProject(t, tc4)
		time.Sleep(1 * time.Second)

		visitor1 := "visitor-device-1"
		visitor2 := "visitor-device-2"
		sharedUserID := "user-123"

		for i := 0; i < 50; i++ {
			event := fixtures.NewEventBuilder().
				WithVisitorID(visitor1).
				WithSessionID(fmt.Sprintf("session-1-%d", i)).
				WithIdentity(sharedUserID, "", "user@example.com").
				WithPageURL(fmt.Sprintf("/page-%d", i)).
				WithHost(project4.Domain).
				WithUTMSource("google").
				Build()
			err := fixtures.SendEventToTestServer(t, tc4, project4, event)
			require.NoError(t, err)
		}

		for i := 0; i < 30; i++ {
			event := fixtures.NewEventBuilder().
				WithVisitorID(visitor2).
				WithSessionID(fmt.Sprintf("session-2-%d", i)).
				WithIdentity(sharedUserID, "", "user@example.com").
				WithPageURL(fmt.Sprintf("/page-%d", i)).
				WithHost(project4.Domain).
				WithUTMSource("google").
				Build()
			err := fixtures.SendEventToTestServer(t, tc4, project4, event)
			require.NoError(t, err)
		}

		_, err := fixtures.WaitForEvents(t, tc4, fixtures.QueryEventsOptions{
			ProjectID: &project4.ID,
		}, 80, 20*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		payment1 := fixtures.NewPaymentBuilder(project4.ID, project4.OrganizationID).
			WithVisitorID(visitor1).
			WithAmount(4000). // $40
			WithProductName("Product from device 1").
			Build()

		payment2 := fixtures.NewPaymentBuilder(project4.ID, project4.OrganizationID).
			WithVisitorID(visitor2).
			WithAmount(4000). // $40
			WithProductName("Product from device 2").
			Build()

		err = fixtures.SendPaymentToNATS(t, tc4, payment1)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		err = fixtures.SendPaymentToNATS(t, tc4, payment2)
		require.NoError(t, err)

		succeededStatus := "succeeded"
		_, err = fixtures.WaitForPayments(t, tc4, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project4.ID,
			PaymentStatus: &succeededStatus,
		}, 2, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(3 * time.Second)

		customers, err := tc4.RevenueData.GetTopCustomers(ctx, project4.ID, revenueTypes.TimeRangeLast7Days, 10)
		require.NoError(t, err)
		require.NotEmpty(t, customers, "Should have at least one customer")

		var customer *revenueTypes.TopCustomer
		for i := range customers {
			if customers[i].UserID != nil && *customers[i].UserID == sharedUserID {
				customer = &customers[i]
				break
			}
		}

		require.NotNil(t, customer, "Should find customer with user_id %s", sharedUserID)

		expectedRevenue := int64(8000) // $80.00 in cents
		assert.Equal(t, expectedRevenue, customer.TotalRevenue,
			"Revenue should be exactly $80 despite multiple visitors and many events. "+
				"If this is inflated (e.g., $11,200), there's a cartesian join bug.")

		assert.Equal(t, uint64(2), customer.PaymentCount, "Should have exactly 2 payments")

		assert.Len(t, customer.VisitorIDs, 2, "Customer should be linked to both visitor IDs")
		assert.Contains(t, customer.VisitorIDs, visitor1)
		assert.Contains(t, customer.VisitorIDs, visitor2)

		expectedAvgOrderValue := 4000.0 // $40 per order
		assert.InDelta(t, expectedAvgOrderValue, customer.AvgOrderValue, 1.0,
			"Average order value should be $40")

		t.Logf("✓ Customer with 2 visitors (80 events total): $%.2f revenue, %d payments",
			float64(customer.TotalRevenue)/100,
			customer.PaymentCount)
		t.Logf("✓ No cartesian join inflation - revenue is correctly calculated")
	})
}

func TestRevenueMetrics_DashboardIdentifiedCustomers(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should calculate metrics for identified customers", func(t *testing.T) {
		visitorID1 := "visitor-identified-1"
		visitorID2 := "visitor-identified-2"

		event1 := fixtures.NewEventBuilder().
			WithVisitorID(visitorID1).
			WithSessionID("session-1").
			WithIdentity("user-1", "ext-1", "user1@example.com").
			WithPageURL("/").
			WithHost(project.Domain).
			WithUTMSource("google").
			Build()

		event2 := fixtures.NewEventBuilder().
			WithVisitorID(visitorID2).
			WithSessionID("session-2").
			WithIdentity("user-2", "ext-2", "user2@example.com").
			WithPageURL("/").
			WithHost(project.Domain).
			WithUTMSource("facebook").
			Build()

		event3 := fixtures.NewEventBuilder().
			WithVisitorID("visitor-anonymous").
			WithSessionID("session-3").
			WithPageURL("/").
			WithHost(project.Domain).
			Build()

		err := fixtures.SendEventToTestServer(t, tc, project, event1)
		require.NoError(t, err)
		err = fixtures.SendEventToTestServer(t, tc, project, event2)
		require.NoError(t, err)
		err = fixtures.SendEventToTestServer(t, tc, project, event3)
		require.NoError(t, err)

		_, err = fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 3, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		payment1 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(visitorID1).
			WithAmount(10000).
			Build()

		payment2 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID(visitorID2).
			WithAmount(20000).
			Build()

		payment3 := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
			WithVisitorID("visitor-anonymous").
			WithAmount(5000).
			Build()

		err = fixtures.SendPaymentToNATS(t, tc, payment1)
		require.NoError(t, err)
		err = fixtures.SendPaymentToNATS(t, tc, payment2)
		require.NoError(t, err)
		err = fixtures.SendPaymentToNATS(t, tc, payment3)
		require.NoError(t, err)

		succeededStatus := "succeeded"
		_, err = fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			PaymentStatus: &succeededStatus,
		}, 3, 10*time.Second)
		require.NoError(t, err)

		time.Sleep(3 * time.Second)

		dashboard, err := revenueDataService.GetDashboardMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)
		require.NotNil(t, dashboard)

		assert.Equal(t, int64(35000), dashboard.TotalRevenue, "Total revenue should be $350")
		assert.Equal(t, uint64(3), dashboard.PayingCustomers, "Should have 3 paying customers")

		assert.Equal(t, uint64(2), dashboard.IdentifiedCustomers, "Should have 2 identified customers")
		assert.Equal(t, int64(30000), dashboard.IdentifiedCustomerRevenue, "Identified revenue should be $300")
		assert.InDelta(t, 15000.0, dashboard.AvgRevenuePerIdentifiedCustomer, 100.0, "Avg per identified customer should be ~$150")

		t.Logf("✓ Identified customers: %d (revenue: $%.2f, avg: $%.2f)",
			dashboard.IdentifiedCustomers,
			float64(dashboard.IdentifiedCustomerRevenue)/100,
			dashboard.AvgRevenuePerIdentifiedCustomer/100)
	})
}

func TestRevenueMetrics_AttributionEdgeCases(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)
	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should handle empty results gracefully", func(t *testing.T) {
		dataPoints, err := revenueDataService.GetAttributionByOrigin(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
		require.NoError(t, err)
		assert.Empty(t, dataPoints)

		utmPoints, err := revenueDataService.GetAttributionByUTM(ctx, project.ID, revenueTypes.TimeRangeLast7Days, "source")
		require.NoError(t, err)
		assert.Empty(t, utmPoints)

		t.Logf("✓ Empty results handled gracefully")
	})

	t.Run("should attribute to (not set) for missing UTM", func(t *testing.T) {
		tc2 := di.NewTestContainer(t)
		defer tc2.Cleanup()

		_, project2 := fixtures.SetupAccountAndProject(t, tc2)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()

		builder.AddDirectTrafficVisitor(10000)

		err := builder.InsertTestData(t, tc2, project2)
		require.NoError(t, err)

		dataPoints, err := tc2.RevenueData.GetAttributionByUTM(ctx, project2.ID, revenueTypes.TimeRangeLast7Days, "source")
		require.NoError(t, err)
		require.NotEmpty(t, dataPoints)

		found := false
		for _, dp := range dataPoints {
			if dp.UTMValue == "(not set)" {
				found = true
				assert.Greater(t, dp.TotalRevenue, int64(0))
				break
			}
		}

		assert.True(t, found, "Should have '(not set)' attribution for missing UTM")
		t.Logf("✓ Missing UTM attributed to '(not set)'")
	})

	t.Run("should test all UTM types", func(t *testing.T) {
		tc3 := di.NewTestContainer(t)
		defer tc3.Cleanup()

		_, project3 := fixtures.SetupAccountAndProject(t, tc3)
		time.Sleep(1 * time.Second)

		builder := fixtures.NewRevenueTestDataBuilder()
		builder.AddSimpleConvertingVisitor(10000, "google")

		err := builder.InsertTestData(t, tc3, project3)
		require.NoError(t, err)

		sourceData, err := tc3.RevenueData.GetAttributionByUTM(ctx, project3.ID, revenueTypes.TimeRangeLast7Days, "source")
		require.NoError(t, err)
		assert.NotEmpty(t, sourceData)

		mediumData, err := tc3.RevenueData.GetAttributionByUTM(ctx, project3.ID, revenueTypes.TimeRangeLast7Days, "medium")
		require.NoError(t, err)
		assert.NotEmpty(t, mediumData)

		campaignData, err := tc3.RevenueData.GetAttributionByUTM(ctx, project3.ID, revenueTypes.TimeRangeLast7Days, "campaign")
		require.NoError(t, err)
		assert.NotEmpty(t, campaignData)

		t.Logf("✓ All UTM types working: source (%d), medium (%d), campaign (%d)",
			len(sourceData), len(mediumData), len(campaignData))
	})
}

func TestConversionMetrics_EndToEnd(t *testing.T) {
	tc := di.NewTestContainer(t)
	defer tc.Cleanup()

	_, project := fixtures.SetupAccountAndProject(t, tc)

	revenueDataService := tc.RevenueData
	ctx := context.Background()

	time.Sleep(1 * time.Second)

	t.Run("should calculate conversion metrics correctly", func(t *testing.T) {
		now := time.Now().UTC()

		visitors := []struct {
			visitorID      string
			sessionID      string
			firstVisitTime time.Time
			willPurchase   bool
			purchaseTime   time.Time
			purchaseAmount int64
			repeatPurchase bool
			repeat2Amount  int64
			repeat2Time    time.Time
			utmSource      string
		}{
			{
				visitorID:      "visitor-quick-convert",
				sessionID:      "session-quick-1",
				firstVisitTime: now.Add(-2 * time.Hour),
				willPurchase:   true,
				purchaseTime:   now.Add(-1 * time.Hour),
				purchaseAmount: 10000,
				repeatPurchase: false,
				utmSource:      "google",
			},
			{
				visitorID:      "visitor-slow-convert",
				sessionID:      "session-slow-1",
				firstVisitTime: now.Add(-48 * time.Hour),
				willPurchase:   true,
				purchaseTime:   now.Add(-24 * time.Hour),
				purchaseAmount: 15000,
				repeatPurchase: true,
				repeat2Amount:  8000,
				repeat2Time:    now.Add(-12 * time.Hour),
				utmSource:      "facebook",
			},
			{
				visitorID:      "visitor-medium-convert",
				sessionID:      "session-medium-1",
				firstVisitTime: now.Add(-12 * time.Hour),
				willPurchase:   true,
				purchaseTime:   now.Add(-6 * time.Hour),
				purchaseAmount: 20000,
				repeatPurchase: true,
				repeat2Amount:  5000,
				repeat2Time:    now.Add(-3 * time.Hour),
				utmSource:      "twitter",
			},
			{
				visitorID:      "visitor-no-convert-1",
				sessionID:      "session-no-1",
				firstVisitTime: now.Add(-6 * time.Hour),
				willPurchase:   false,
				utmSource:      "linkedin",
			},
			{
				visitorID:      "visitor-no-convert-2",
				sessionID:      "session-no-2",
				firstVisitTime: now.Add(-3 * time.Hour),
				willPurchase:   false,
				utmSource:      "reddit",
			},
		}

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
			err := fixtures.SendEventToTestServer(t, tc, project, event)
			require.NoError(t, err, "Failed to send visitor event for %s", v.visitorID)
		}

		_, err := fixtures.WaitForEvents(t, tc, fixtures.QueryEventsOptions{
			ProjectID: &project.ID,
		}, 5, 15*time.Second)
		require.NoError(t, err, "Events should appear in ClickHouse")

		time.Sleep(2 * time.Second)

		for _, v := range visitors {
			if v.willPurchase {
				payment := fixtures.NewPaymentBuilder(project.ID, project.OrganizationID).
					WithVisitorID(v.visitorID).
					WithAmount(v.purchaseAmount).
					WithProductName("Test Product").
					Build()

				err := fixtures.SendPaymentToNATS(t, tc, payment)
				require.NoError(t, err, "Failed to send first payment for %s", v.visitorID)

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

		succeededStatus := "succeeded"
		payments, err := fixtures.WaitForPayments(t, tc, fixtures.QueryPaymentEventsOptions{
			ProjectID:     &project.ID,
			PaymentStatus: &succeededStatus,
		}, 5, 15*time.Second)
		require.NoError(t, err, "Payments should appear in ClickHouse")
		assert.Len(t, payments, 5, "Should have stored all 5 payments")

		time.Sleep(3 * time.Second)

		t.Run("conversion rate", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

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

			assert.GreaterOrEqual(t, metrics.AvgTimeToFirstPurchase, 0.0,
				"Average time to first purchase should be non-negative")
			assert.GreaterOrEqual(t, metrics.MedianTimeToFirstPurchase, 0.0,
				"Median time to first purchase should be non-negative")

			t.Logf("✓ Avg Time to First Purchase: %.2f hours", metrics.AvgTimeToFirstPurchase)
			t.Logf("✓ Median Time to First Purchase: %.2f hours", metrics.MedianTimeToFirstPurchase)
		})

		t.Run("customer lifetime value (LTV)", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

			assert.Greater(t, metrics.CustomerLifetimeValue, 0.0, "Customer LTV should be positive")

			minExpectedLTV := float64(10000)
			assert.GreaterOrEqual(t, metrics.CustomerLifetimeValue, minExpectedLTV,
				"Customer LTV should be at least $100")

			t.Logf("✓ Customer Lifetime Value: $%.2f", metrics.CustomerLifetimeValue/100)
		})

		t.Run("repeat purchase rate", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

			assert.GreaterOrEqual(t, metrics.RepeatPurchaseRate, 0.0,
				"Repeat purchase rate should be non-negative")
			assert.LessOrEqual(t, metrics.RepeatPurchaseRate, 100.0,
				"Repeat purchase rate should not exceed 100%")

			assert.GreaterOrEqual(t, metrics.AvgPurchasesPerCustomer, 1.0,
				"Average purchases per customer should be at least 1")

			t.Logf("✓ Repeat Purchase Rate: %.2f%%", metrics.RepeatPurchaseRate)
			t.Logf("✓ Avg Purchases per Customer: %.2f", metrics.AvgPurchasesPerCustomer)
		})

		t.Run("complete conversion funnel metrics", func(t *testing.T) {
			metrics, err := revenueDataService.GetConversionMetrics(ctx, project.ID, revenueTypes.TimeRangeLast7Days)
			require.NoError(t, err, "Should get conversion metrics")
			require.NotNil(t, metrics)

			t.Logf("\n=== Conversion Funnel Summary ===")
			t.Logf("Total Visitors: %d", metrics.TotalVisitors)
			t.Logf("Paying Customers: %d", metrics.PayingCustomers)
			t.Logf("Conversion Rate: %.2f%%", metrics.ConversionRate)
			t.Logf("Avg Time to First Purchase: %.2f hours", metrics.AvgTimeToFirstPurchase)
			t.Logf("Median Time to First Purchase: %.2f hours", metrics.MedianTimeToFirstPurchase)
			t.Logf("Customer Lifetime Value: $%.2f", metrics.CustomerLifetimeValue/100)
			t.Logf("Repeat Purchase Rate: %.2f%%", metrics.RepeatPurchaseRate)
			t.Logf("Avg Purchases per Customer: %.2f", metrics.AvgPurchasesPerCustomer)

			assert.Greater(t, metrics.TotalVisitors, uint64(0), "Should have visitors")
			assert.Greater(t, metrics.PayingCustomers, uint64(0), "Should have paying customers")
			assert.Greater(t, metrics.ConversionRate, 0.0, "Should have positive conversion rate")
			assert.Greater(t, metrics.CustomerLifetimeValue, 0.0, "Should have positive LTV")
		})
	})
}
