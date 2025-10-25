package data

import (
	"context"
	"fmt"
	"time"
	"zori/internal/storage/clickhouse"
	"zori/services/revenue/types"
	ingestionData "zori/services/ingestion/data"
)

type RevenueData struct {
	clickDb           *clickhouse.ClickhouseDB
	visitorRepository *ingestionData.VisitorRepository
}

func NewRevenueData(clickDb *clickhouse.ClickhouseDB, visitorRepository *ingestionData.VisitorRepository) *RevenueData {
	return &RevenueData{
		clickDb:           clickDb,
		visitorRepository: visitorRepository,
	}
}

// GetTimeRangeBounds returns the start time and interval for a given time range
func GetTimeRangeBounds(timeRange types.TimeRange) (time.Time, string, error) {
	now := time.Now().UTC()

	switch timeRange {
	case types.TimeRangeLastHour:
		return now.Add(-1 * time.Hour), "toStartOfMinute", nil
	case types.TimeRangeToday:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), "toStartOfHour", nil
	case types.TimeRangeLast7Days:
		return now.AddDate(0, 0, -7), "toStartOfHour", nil
	case types.TimeRangeLast30Days:
		return now.AddDate(0, 0, -30), "toStartOfDay", nil
	case types.TimeRangeLast90Days:
		return now.AddDate(0, 0, -90), "toStartOfDay", nil
	default:
		return time.Time{}, "", fmt.Errorf("invalid time range: %s", timeRange)
	}
}

// GetDashboardMetrics returns key revenue metrics for the dashboard
func (r *RevenueData) GetDashboardMetrics(ctx context.Context, projectID string, timeRange types.TimeRange) (*types.DashboardResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	// Main dashboard query combining multiple metrics
	query := `
		WITH payment_data AS (
			SELECT
				amount,
				visitor_id,
				currency,
				payment_id
			FROM payment_events
			WHERE project_id = ?
				AND payment_timestamp_utc >= ?
				AND payment_timestamp_utc <= now()
				AND payment_status = 'succeeded'
		),
		visitor_data AS (
			SELECT DISTINCT visitor_id
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
		),
		session_data AS (
			SELECT COUNT(DISTINCT session_id) as unique_sessions
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
		),
		identified_customers AS (
			SELECT DISTINCT p.visitor_id
			FROM payment_data p
			INNER JOIN events e ON p.visitor_id = e.visitor_id
			WHERE e.project_id = ?
				AND (e.user_id IS NOT NULL OR e.external_id IS NOT NULL OR e.email_hash IS NOT NULL)
		)
		SELECT
			-- Core revenue
			COALESCE(SUM(pd.amount), 0) as total_revenue,
			COUNT(DISTINCT pd.payment_id) as total_payments,
			any(pd.currency) as currency,

			-- Customer metrics
			COUNT(DISTINCT pd.visitor_id) as paying_customers,
			CASE
				WHEN (SELECT COUNT(*) FROM visitor_data) > 0
				THEN COUNT(DISTINCT pd.visitor_id) * 100.0 / (SELECT COUNT(*) FROM visitor_data)
				ELSE 0
			END as conversion_rate,

			-- Average metrics
			CASE
				WHEN COUNT(DISTINCT pd.payment_id) > 0
				THEN COALESCE(SUM(pd.amount), 0) / COUNT(DISTINCT pd.payment_id)
				ELSE 0
			END as avg_order_value,
			CASE
				WHEN COUNT(DISTINCT pd.visitor_id) > 0
				THEN COALESCE(SUM(pd.amount), 0) / COUNT(DISTINCT pd.visitor_id)
				ELSE 0
			END as avg_revenue_per_customer,
			CASE
				WHEN (SELECT unique_sessions FROM session_data) > 0
				THEN COALESCE(SUM(pd.amount), 0) / (SELECT unique_sessions FROM session_data)
				ELSE 0
			END as avg_revenue_per_session,

			-- Identified customers
			(SELECT COUNT(*) FROM identified_customers) as identified_customers,
			COALESCE(SUM(CASE WHEN pd.visitor_id IN (SELECT visitor_id FROM identified_customers) THEN pd.amount ELSE 0 END), 0) as identified_customer_revenue,
			CASE
				WHEN (SELECT COUNT(*) FROM identified_customers) > 0
				THEN COALESCE(SUM(CASE WHEN pd.visitor_id IN (SELECT visitor_id FROM identified_customers) THEN pd.amount ELSE 0 END), 0) / (SELECT COUNT(*) FROM identified_customers)
				ELSE 0
			END as avg_revenue_per_identified_customer
		FROM payment_data pd
	`

	var response types.DashboardResponse
	row := r.clickDb.Db().QueryRow(ctx, query,
		projectID, startTime, // payment_data
		projectID, startTime, // visitor_data
		projectID, startTime, // session_data
		projectID, // identified_customers
	)

	if err := row.Scan(
		&response.TotalRevenue,
		&response.TotalPayments,
		&response.Currency,
		&response.PayingCustomers,
		&response.ConversionRate,
		&response.AvgOrderValue,
		&response.AvgRevenuePerCustomer,
		&response.AvgRevenuePerSession,
		&response.IdentifiedCustomers,
		&response.IdentifiedCustomerRevenue,
		&response.AvgRevenuePerIdentifiedCustomer,
	); err != nil {
		return nil, fmt.Errorf("failed to get dashboard metrics: %w", err)
	}

	if response.Currency == "" {
		response.Currency = "USD"
	}

	return &response, nil
}

// GetAttributionByOrigin returns revenue attributed to traffic origins using first-touch attribution
func (r *RevenueData) GetAttributionByOrigin(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.OriginAttributionDataPoint, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := `
		WITH visitors_in_period AS (
			SELECT DISTINCT visitor_id
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
		),
		revenue_totals AS (
			SELECT COALESCE(SUM(amount), 0) as total_revenue
			FROM payment_events
			WHERE project_id = ?
				AND payment_timestamp_utc >= ?
				AND payment_timestamp_utc <= now()
				AND payment_status = 'succeeded'
		),
		visitor_origins AS (
			SELECT
				ft.visitor_id,
				CASE
					WHEN argMinMerge(ft.first_referrer_domain) IS NULL OR argMinMerge(ft.first_referrer_domain) = '' THEN 'Direct'
					ELSE argMinMerge(ft.first_referrer_domain)
				END as origin
			FROM visitor_first_touch_attribution ft
			INNER JOIN visitors_in_period vip ON ft.visitor_id = vip.visitor_id
			WHERE ft.project_id = ?
			GROUP BY ft.visitor_id
		),
		origin_stats AS (
			SELECT
				vo.origin,
				uniq(vo.visitor_id) as unique_visitors,
				COALESCE(SUM(p.amount), 0) as total_revenue,
				uniq(p.visitor_id) as paying_customers,
				countIf(p.payment_id IS NOT NULL) as payment_count,
				any(p.currency) as currency
			FROM visitor_origins vo
			LEFT JOIN payment_events p
				ON vo.visitor_id = p.visitor_id
				AND p.payment_status = 'succeeded'
				AND p.project_id = ?
				AND p.payment_timestamp_utc >= ?
				AND p.payment_timestamp_utc <= now()
			GROUP BY vo.origin
		)
		SELECT
			os.origin,
			os.total_revenue,
			CASE
				WHEN rt.total_revenue > 0
				THEN os.total_revenue * 100.0 / rt.total_revenue
				ELSE 0
			END as revenue_percentage,
			os.paying_customers,
			os.unique_visitors,
			CASE
				WHEN os.unique_visitors > 0
				THEN os.paying_customers * 100.0 / os.unique_visitors
				ELSE 0
			END as conversion_rate,
			CASE
				WHEN os.paying_customers > 0
				THEN os.total_revenue / os.paying_customers
				ELSE 0
			END as avg_revenue_per_customer,
			os.payment_count,
			os.currency
		FROM origin_stats os
		CROSS JOIN revenue_totals rt
		ORDER BY os.total_revenue DESC
		LIMIT 20
	`

	rows, err := r.clickDb.Db().Query(ctx, query,
		projectID, startTime, // visitors_in_period
		projectID, startTime, // revenue_totals
		projectID,            // visitor_origins
		projectID, startTime, // origin_stats payment join
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query attribution by origin: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.OriginAttributionDataPoint
	for rows.Next() {
		var dp types.OriginAttributionDataPoint
		if err := rows.Scan(
			&dp.Origin,
			&dp.TotalRevenue,
			&dp.RevenuePercentage,
			&dp.PayingCustomers,
			&dp.UniqueVisitors,
			&dp.ConversionRate,
			&dp.AvgRevenuePerCustomer,
			&dp.PaymentCount,
			&dp.Currency,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		dataPoints = append(dataPoints, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return dataPoints, nil
}

// GetAttributionByUTM returns revenue attributed to UTM parameters using first-touch attribution
func (r *RevenueData) GetAttributionByUTM(ctx context.Context, projectID string, timeRange types.TimeRange, utmType string) ([]types.UTMAttributionDataPoint, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	// Determine which UTM field to use from the attribution table
	utmField := "first_utm_source"
	switch utmType {
	case "medium":
		utmField = "first_utm_medium"
	case "campaign":
		utmField = "first_utm_campaign"
	case "source":
		utmField = "first_utm_source"
	default:
		utmField = "first_utm_source"
	}

	query := fmt.Sprintf(`
		WITH revenue_totals AS (
			SELECT SUM(amount) as total_revenue
			FROM payment_events
			WHERE project_id = ?
				AND payment_timestamp_utc >= ?
				AND payment_timestamp_utc <= now()
				AND payment_status = 'succeeded'
		),
		payments_in_period AS (
			SELECT
				visitor_id,
				amount,
				payment_id,
				currency
			FROM payment_events
			WHERE project_id = ?
				AND payment_timestamp_utc >= ?
				AND payment_timestamp_utc <= now()
				AND payment_status = 'succeeded'
		),
		visitor_utm AS (
			SELECT
				ft.visitor_id,
				CASE
					WHEN argMinMerge(ft.%s) IS NULL OR argMinMerge(ft.%s) = '' THEN '(not set)'
					ELSE argMinMerge(ft.%s)
				END as utm_value
			FROM visitor_first_touch_attribution ft
			INNER JOIN payments_in_period pip ON ft.visitor_id = pip.visitor_id
			WHERE ft.project_id = ?
			GROUP BY ft.visitor_id
		),
		visitors_in_period AS (
			SELECT DISTINCT visitor_id
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
		),
		all_visitor_utm AS (
			SELECT
				ft.visitor_id,
				CASE
					WHEN argMinMerge(ft.%s) IS NULL OR argMinMerge(ft.%s) = '' THEN '(not set)'
					ELSE argMinMerge(ft.%s)
				END as utm_value
			FROM visitor_first_touch_attribution ft
			INNER JOIN visitors_in_period vip ON ft.visitor_id = vip.visitor_id
			WHERE ft.project_id = ?
			GROUP BY ft.visitor_id
		)
		SELECT
			vu.utm_value,
			COALESCE(SUM(p.amount), 0) as total_revenue,
			CASE
				WHEN (SELECT total_revenue FROM revenue_totals) > 0
				THEN (COALESCE(SUM(p.amount), 0) * 100.0 / (SELECT total_revenue FROM revenue_totals))
				ELSE 0
			END as revenue_percentage,
			uniq(p.visitor_id) as paying_customers,
			(SELECT uniq(visitor_id) FROM all_visitor_utm WHERE utm_value = vu.utm_value) as unique_visitors,
			CASE
				WHEN (SELECT uniq(visitor_id) FROM all_visitor_utm WHERE utm_value = vu.utm_value) > 0
				THEN (uniq(p.visitor_id) * 100.0 / (SELECT uniq(visitor_id) FROM all_visitor_utm WHERE utm_value = vu.utm_value))
				ELSE 0
			END as conversion_rate,
			CASE
				WHEN uniq(p.visitor_id) > 0
				THEN COALESCE(SUM(p.amount), 0) / uniq(p.visitor_id)
				ELSE 0
			END as avg_revenue_per_customer,
			countIf(p.payment_id IS NOT NULL) as payment_count,
			any(p.currency) as currency
		FROM visitor_utm vu
		LEFT JOIN payments_in_period p ON vu.visitor_id = p.visitor_id
		GROUP BY vu.utm_value
		ORDER BY total_revenue DESC
		LIMIT 20
	`, utmField, utmField, utmField, utmField, utmField, utmField)

	rows, err := r.clickDb.Db().Query(ctx, query,
		projectID, startTime, // revenue_totals
		projectID, startTime, // payments_in_period
		projectID,            // visitor_utm
		projectID, startTime, // visitors_in_period
		projectID, // all_visitor_utm
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query attribution by UTM: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.UTMAttributionDataPoint
	for rows.Next() {
		var dp types.UTMAttributionDataPoint
		if err := rows.Scan(
			&dp.UTMValue,
			&dp.TotalRevenue,
			&dp.RevenuePercentage,
			&dp.PayingCustomers,
			&dp.UniqueVisitors,
			&dp.ConversionRate,
			&dp.AvgRevenuePerCustomer,
			&dp.PaymentCount,
			&dp.Currency,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		dataPoints = append(dataPoints, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return dataPoints, nil
}

// GetTimeline returns revenue over time using materialized views
func (r *RevenueData) GetTimeline(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.TimelineDataPoint, error) {
	startTime, intervalFunc, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	var query string

	if timeRange == types.TimeRangeLast30Days || timeRange == types.TimeRangeLast90Days {
		// Use daily materialized view for longer ranges
		query = `
			SELECT
				time_bucket,
				SUM(total_revenue) as total_revenue,
				SUM(payment_count) as payment_count,
				any(currency) as currency
			FROM revenue_timeline_daily_mv
			WHERE project_id = ?
				AND time_bucket >= toDate(?)
				AND time_bucket <= today()
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`
	} else if timeRange == types.TimeRangeToday || timeRange == types.TimeRangeLast7Days {
		// Use hourly materialized view for shorter ranges
		query = `
			SELECT
				time_bucket,
				SUM(total_revenue) as total_revenue,
				SUM(payment_count) as payment_count,
				any(currency) as currency
			FROM revenue_timeline_hourly_mv
			WHERE project_id = ?
				AND time_bucket >= ?
				AND time_bucket <= now()
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`
	} else {
		// Fall back to raw query for sub-hour granularity
		query = fmt.Sprintf(`
			SELECT
				%s(payment_timestamp_utc) as time_bucket,
				COALESCE(SUM(amount), 0) as total_revenue,
				COUNT(*) as payment_count,
				any(currency) as currency
			FROM payment_events
			WHERE project_id = ?
				AND payment_timestamp_utc >= ?
				AND payment_timestamp_utc <= now()
				AND payment_status = 'succeeded'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, intervalFunc)
	}

	rows, err := r.clickDb.Db().Query(ctx, query, projectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue timeline: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.TimelineDataPoint
	for rows.Next() {
		var dp types.TimelineDataPoint
		if err := rows.Scan(&dp.Timestamp, &dp.TotalRevenue, &dp.PaymentCount, &dp.Currency); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		dataPoints = append(dataPoints, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return dataPoints, nil
}

// GetTopCustomers returns the highest revenue customers
func (r *RevenueData) GetTopCustomers(ctx context.Context, projectID string, timeRange types.TimeRange, limit int) ([]types.TopCustomer, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT
			p.visitor_id,
			COALESCE(SUM(p.amount), 0) as total_revenue,
			COUNT(*) as payment_count,
			MIN(p.payment_timestamp_utc) as first_payment_date,
			MAX(p.payment_timestamp_utc) as last_payment_date,
			CASE
				WHEN COUNT(*) > 0
				THEN COALESCE(SUM(p.amount), 0) / COUNT(*)
				ELSE 0
			END as avg_order_value,
			any(p.currency) as currency,
			any(e.location_country_iso) as location_country_iso
		FROM payment_events p
		LEFT JOIN events e ON p.visitor_id = e.visitor_id AND e.project_id = ?
		WHERE p.project_id = ?
			AND p.payment_timestamp_utc >= ?
			AND p.payment_timestamp_utc <= now()
			AND p.payment_status = 'succeeded'
		GROUP BY p.visitor_id
		ORDER BY total_revenue DESC
		LIMIT ?
	`

	rows, err := r.clickDb.Db().Query(ctx, query, projectID, projectID, startTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top customers: %w", err)
	}
	defer rows.Close()

	var customers []types.TopCustomer
	for rows.Next() {
		var customer types.TopCustomer
		if err := rows.Scan(
			&customer.VisitorID,
			&customer.TotalRevenue,
			&customer.PaymentCount,
			&customer.FirstPaymentDate,
			&customer.LastPaymentDate,
			&customer.AvgOrderValue,
			&customer.Currency,
			&customer.LocationCountryISO,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Fetch visitor identity from PostgreSQL
		visitorIdentity, err := r.visitorRepository.GetVisitorByID(ctx, customer.VisitorID)
		if err == nil && visitorIdentity != nil {
			customer.UserID = visitorIdentity.UserID
			customer.ExternalID = visitorIdentity.ExternalID
			customer.Email = visitorIdentity.Email
			customer.Name = visitorIdentity.Name
		}

		// Get first traffic origin
		originQuery := `
			SELECT argMinMerge(first_referrer_domain) as first_origin
			FROM visitor_first_touch_attribution
			WHERE project_id = ? AND visitor_id = ?
			GROUP BY visitor_id
		`
		originRow := r.clickDb.Db().QueryRow(ctx, originQuery, projectID, customer.VisitorID)
		if err := originRow.Scan(&customer.FirstTrafficOrigin); err != nil {
			customer.FirstTrafficOrigin = nil
		}

		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return customers, nil
}

// GetCustomerProfile returns detailed revenue profile for a specific customer
func (r *RevenueData) GetCustomerProfile(ctx context.Context, projectID string, visitorID string) (*types.CustomerProfileResponse, error) {
	var profile types.CustomerProfileResponse
	profile.VisitorID = visitorID

	// Get revenue summary
	summaryQuery := `
		SELECT
			COALESCE(SUM(amount), 0) as total_revenue,
			COUNT(*) as payment_count,
			MIN(payment_timestamp_utc) as first_payment,
			MAX(payment_timestamp_utc) as last_payment,
			CASE
				WHEN COUNT(*) > 0
				THEN COALESCE(SUM(amount), 0) / COUNT(*)
				ELSE 0
			END as avg_order_value,
			any(currency) as currency
		FROM payment_events
		WHERE project_id = ?
			AND visitor_id = ?
			AND payment_status = 'succeeded'
	`

	row := r.clickDb.Db().QueryRow(ctx, summaryQuery, projectID, visitorID)
	if err := row.Scan(
		&profile.TotalRevenue,
		&profile.PaymentCount,
		&profile.FirstPaymentDate,
		&profile.LastPaymentDate,
		&profile.AvgOrderValue,
		&profile.Currency,
	); err != nil {
		// Set defaults if no payments found
		profile.TotalRevenue = 0
		profile.PaymentCount = 0
		profile.AvgOrderValue = 0
		profile.FirstPaymentDate = nil
		profile.LastPaymentDate = nil
	}

	// Get attribution data
	attributionQuery := `
		SELECT
			argMinMerge(first_referrer_domain) as first_origin,
			argMinMerge(first_utm_source) as first_utm_source,
			argMinMerge(first_utm_medium) as first_utm_medium,
			argMinMerge(first_utm_campaign) as first_utm_campaign
		FROM visitor_first_touch_attribution
		WHERE project_id = ? AND visitor_id = ?
		GROUP BY visitor_id
	`

	attrRow := r.clickDb.Db().QueryRow(ctx, attributionQuery, projectID, visitorID)
	if err := attrRow.Scan(
		&profile.FirstTrafficOrigin,
		&profile.FirstUTMSource,
		&profile.FirstUTMMedium,
		&profile.FirstUTMCampaign,
	); err != nil {
		// It's okay if there's no attribution data
		profile.FirstTrafficOrigin = nil
		profile.FirstUTMSource = nil
		profile.FirstUTMMedium = nil
		profile.FirstUTMCampaign = nil
	}

	// Get payment history
	paymentsQuery := `
		SELECT
			payment_id,
			amount,
			currency,
			payment_status,
			product_name,
			payment_timestamp_utc,
			provider_type
		FROM payment_events
		WHERE project_id = ?
			AND visitor_id = ?
		ORDER BY payment_timestamp_utc DESC
		LIMIT 50
	`

	paymentsRows, err := r.clickDb.Db().Query(ctx, paymentsQuery, projectID, visitorID)
	if err == nil {
		defer paymentsRows.Close()

		var payments []types.Payment
		for paymentsRows.Next() {
			var payment types.Payment
			if err := paymentsRows.Scan(
				&payment.PaymentID,
				&payment.Amount,
				&payment.Currency,
				&payment.Status,
				&payment.ProductName,
				&payment.PaymentTimestamp,
				&payment.ProviderType,
			); err == nil {
				payments = append(payments, payment)
			}
		}
		profile.Payments = payments
	}

	// Get revenue over time (last 90 days)
	now := time.Now().UTC()
	revenueStartTime := now.AddDate(0, 0, -90)

	revenueOverTimeQuery := `
		SELECT
			toStartOfDay(payment_timestamp_utc) as time_bucket,
			COALESCE(SUM(amount), 0) as total_revenue,
			COUNT(*) as payment_count
		FROM payment_events
		WHERE project_id = ?
			AND visitor_id = ?
			AND payment_timestamp_utc >= ?
			AND payment_timestamp_utc <= now()
			AND payment_status = 'succeeded'
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`

	timeRows, err := r.clickDb.Db().Query(ctx, revenueOverTimeQuery, projectID, visitorID, revenueStartTime)
	if err == nil {
		defer timeRows.Close()

		var revenueOverTime []types.RevenueOverTimeDataPoint
		for timeRows.Next() {
			var dp types.RevenueOverTimeDataPoint
			if err := timeRows.Scan(&dp.Timestamp, &dp.TotalRevenue, &dp.PaymentCount); err == nil {
				revenueOverTime = append(revenueOverTime, dp)
			}
		}
		profile.RevenueOverTime = revenueOverTime
	}

	// Fetch visitor identity from PostgreSQL
	visitorIdentity, err := r.visitorRepository.GetVisitorByID(ctx, visitorID)
	if err == nil && visitorIdentity != nil {
		profile.UserID = visitorIdentity.UserID
		profile.ExternalID = visitorIdentity.ExternalID
		profile.Email = visitorIdentity.Email
		profile.Name = visitorIdentity.Name
	}

	return &profile, nil
}

// GetConversionMetrics returns conversion funnel and customer value metrics
func (r *RevenueData) GetConversionMetrics(ctx context.Context, projectID string, timeRange types.TimeRange) (*types.ConversionMetricsResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := `
		WITH visitor_data AS (
			SELECT
				visitor_id,
				MIN(client_timestamp_utc) as first_visit
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
			GROUP BY visitor_id
		),
		payment_data AS (
			SELECT
				visitor_id,
				MIN(payment_timestamp_utc) as first_payment,
				COUNT(*) as payment_count,
				SUM(amount) as total_revenue
			FROM payment_events
			WHERE project_id = ?
				AND payment_timestamp_utc >= ?
				AND payment_timestamp_utc <= now()
				AND payment_status = 'succeeded'
			GROUP BY visitor_id
		)
		SELECT
			COUNT(DISTINCT vd.visitor_id) as total_visitors,
			COUNT(DISTINCT pd.visitor_id) as paying_customers,
			CASE
				WHEN COUNT(DISTINCT vd.visitor_id) > 0
				THEN COUNT(DISTINCT pd.visitor_id) * 100.0 / COUNT(DISTINCT vd.visitor_id)
				ELSE 0
			END as conversion_rate,
			AVG(dateDiff('hour', vd.first_visit, pd.first_payment)) as avg_time_to_purchase,
			median(dateDiff('hour', vd.first_visit, pd.first_payment)) as median_time_to_purchase,
			CASE
				WHEN COUNT(DISTINCT pd.visitor_id) > 0
				THEN SUM(pd.total_revenue) / COUNT(DISTINCT pd.visitor_id)
				ELSE 0
			END as customer_lifetime_value,
			CASE
				WHEN COUNT(DISTINCT pd.visitor_id) > 0
				THEN countIf(pd.payment_count > 1) * 100.0 / COUNT(DISTINCT pd.visitor_id)
				ELSE 0
			END as repeat_purchase_rate,
			AVG(pd.payment_count) as avg_purchases_per_customer
		FROM visitor_data vd
		LEFT JOIN payment_data pd ON vd.visitor_id = pd.visitor_id
	`

	var response types.ConversionMetricsResponse
	row := r.clickDb.Db().QueryRow(ctx, query,
		projectID, startTime,
		projectID, startTime,
	)

	if err := row.Scan(
		&response.TotalVisitors,
		&response.PayingCustomers,
		&response.ConversionRate,
		&response.AvgTimeToFirstPurchase,
		&response.MedianTimeToFirstPurchase,
		&response.CustomerLifetimeValue,
		&response.RepeatPurchaseRate,
		&response.AvgPurchasesPerCustomer,
	); err != nil {
		return nil, fmt.Errorf("failed to get conversion metrics: %w", err)
	}

	return &response, nil
}
