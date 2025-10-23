package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/types"
	ingestionData "zori/services/ingestion/data"

	"golang.org/x/sync/errgroup"
)

type AnalyticsData struct {
	clickDb           *clickhouse.ClickhouseDB
	visitorRepository *ingestionData.VisitorRepository
}

func NewAnalyticsData(clickDb *clickhouse.ClickhouseDB, visitorRepository *ingestionData.VisitorRepository) *AnalyticsData {
	return &AnalyticsData{
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

// GetVisitorsByDevice returns visitor counts grouped by device type over time
func (a *AnalyticsData) GetVisitorsByDevice(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.VisitorDataPoint, error) {
	startTime, intervalFunc, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT
			%s(client_timestamp_utc) as time_bucket,
			countIf(device_type = 'Mobile') as mobile,
			countIf(device_type = 'Desktop') as desktop,
			countIf(device_type IS NULL OR (device_type != 'Mobile' AND device_type != 'Desktop')) as unknown
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`, intervalFunc)

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitors by device: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.VisitorDataPoint
	for rows.Next() {
		var dp types.VisitorDataPoint
		if err := rows.Scan(&dp.Timestamp, &dp.Mobile, &dp.Desktop, &dp.Unknown); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		dataPoints = append(dataPoints, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return dataPoints, nil
}

// GetUniqueVisitorsByOrigin returns unique visitor counts grouped by traffic origin with revenue data using first-touch attribution
// This endpoint shows visitors who arrived in the time period AND revenue from payments made in the time period (attributed to origin)
func (a *AnalyticsData) GetUniqueVisitorsByOrigin(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.OriginDataPoint, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	// Step 1: Get all visitors who had events in the time period
	// Step 2: Get their first-touch attribution (from whenever they first arrived)
	// Step 3: Get ALL payments made in the time period, attributed to visitor's original source
	query := `
		WITH visitors_in_period AS (
			SELECT DISTINCT visitor_id
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
		),
		totals AS (
			SELECT COUNT(DISTINCT visitor_id) as total_visitors
			FROM visitors_in_period
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
				uniq(p.visitor_id) as paying_visitors,
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
			os.unique_visitors,
			os.unique_visitors * 100.0 / t.total_visitors as percentage,
			os.total_revenue,
			CASE
				WHEN rt.total_revenue > 0
				THEN os.total_revenue * 100.0 / rt.total_revenue
				ELSE 0
			END as revenue_percentage,
			os.paying_visitors,
			CASE
				WHEN os.unique_visitors > 0
				THEN os.paying_visitors * 100.0 / os.unique_visitors
				ELSE 0
			END as conversion_rate,
			CASE
				WHEN os.paying_visitors > 0
				THEN os.total_revenue / os.paying_visitors
				ELSE 0
			END as avg_revenue_per_visitor,
			os.payment_count,
			os.currency
		FROM origin_stats os
		CROSS JOIN totals t
		CROSS JOIN revenue_totals rt
		ORDER BY os.total_revenue DESC
		LIMIT 20
	`

	rows, err := a.clickDb.Db().Query(ctx, query,
		projectID, startTime, // visitors_in_period CTE
		projectID, startTime, // revenue_totals CTE
		projectID,            // visitor_origins CTE
		projectID, startTime, // origin_stats CTE payment join
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitors by origin: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.OriginDataPoint
	for rows.Next() {
		var dp types.OriginDataPoint
		if err := rows.Scan(
			&dp.Origin,
			&dp.UniqueVisitors,
			&dp.Percentage,
			&dp.TotalRevenue,
			&dp.RevenuePercentage,
			&dp.PayingVisitors,
			&dp.ConversionRate,
			&dp.AvgRevenuePerVisitor,
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

// GetUniqueVisitorsByCountry returns unique visitor counts grouped by country
func (a *AnalyticsData) GetUniqueVisitorsByCountry(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.CountryDataPoint, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := `
		WITH totals AS (
			SELECT uniq(visitor_id) as total_visitors
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
		)
		SELECT
			CASE
				WHEN location_country_iso IS NULL OR location_country_iso = '' THEN 'Unknown'
				ELSE location_country_iso
			END as country_code,
			uniq(visitor_id) as unique_visitors,
			(uniq(visitor_id) * 100.0 / (SELECT total_visitors FROM totals)) as percentage
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY country_code
		ORDER BY unique_visitors DESC
		LIMIT 50
	`

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime, projectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitors by country: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.CountryDataPoint
	for rows.Next() {
		var dp types.CountryDataPoint
		if err := rows.Scan(&dp.CountryCode, &dp.UniqueVisitors, &dp.Percentage); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		dataPoints = append(dataPoints, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return dataPoints, nil
}

// GetRecentEvents returns the most recent events for a project
func (a *AnalyticsData) GetRecentEvents(ctx context.Context, projectID string, limit int) ([]types.RecentEvent, error) {
	if limit <= 0 {
		limit = 15
	}

	query := `
		SELECT
			event_name,
			visitor_id,
			client_timestamp_utc,
			page_url,
			page_path,
			referrer_url,
			device_type,
			browser_name,
			location_country_iso,
			location_city
		FROM events
		WHERE project_id = ?
		ORDER BY client_timestamp_utc DESC
		LIMIT ?
	`

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent events: %w", err)
	}
	defer rows.Close()

	var events []types.RecentEvent
	for rows.Next() {
		var event types.RecentEvent
		if err := rows.Scan(
			&event.EventName,
			&event.VisitorID,
			&event.ClientTimestampUTC,
			&event.PageURL,
			&event.PagePath,
			&event.ReferrerURL,
			&event.DeviceType,
			&event.BrowserName,
			&event.LocationCountryISO,
			&event.LocationCity,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return events, nil
}

func (a *AnalyticsData) GetTopVisitors(ctx context.Context, projectID string, timeRange types.TimeRange, limit int) ([]types.TopVisitor, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT
			e.visitor_id,
			count() as event_count,
			max(e.client_timestamp_utc) as last_seen,
			min(e.client_timestamp_utc) as first_seen,
			any(e.location_country_iso) as location_country_iso,
			any(e.location_city) as location_city,
			any(e.device_type) as device_type,
			any(e.browser_name) as browser_name,
			COALESCE(SUM(p.amount), 0) as total_revenue,
			countIf(p.payment_id IS NOT NULL) as payment_count,
			any(p.currency) as currency
		FROM events e
		LEFT JOIN payment_events p
			ON e.visitor_id = p.visitor_id
			AND p.payment_status = 'succeeded'
			AND p.project_id = ?
			AND p.payment_timestamp_utc >= ?
			AND p.payment_timestamp_utc <= now()
		WHERE e.project_id = ?
			AND e.client_timestamp_utc >= ?
			AND e.client_timestamp_utc <= now()
		GROUP BY e.visitor_id
		ORDER BY event_count DESC
		LIMIT ?
	`

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime, projectID, startTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top visitors: %w", err)
	}
	defer rows.Close()

	var visitors []types.TopVisitor
	for rows.Next() {
		var visitor types.TopVisitor
		if err := rows.Scan(
			&visitor.VisitorID,
			&visitor.EventCount,
			&visitor.LastSeen,
			&visitor.FirstSeen,
			&visitor.LocationCountryISO,
			&visitor.LocationCity,
			&visitor.DeviceType,
			&visitor.BrowserName,
			&visitor.TotalRevenue,
			&visitor.PaymentCount,
			&visitor.Currency,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		visitors = append(visitors, visitor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return visitors, nil
}

// GetVisitorProfile returns detailed profile for a single visitor
func (a *AnalyticsData) GetVisitorProfile(ctx context.Context, projectID string, visitorID string) (*types.VisitorProfileResponse, error) {
	// Get visitor summary stats
	summaryQuery := `
		SELECT
			visitor_id,
			min(client_timestamp_utc) as first_seen,
			max(client_timestamp_utc) as last_seen,
			count() as total_events,
			any(location_country_iso) as location_country_iso,
			any(location_city) as location_city
		FROM events
		WHERE project_id = ?
			AND visitor_id = ?
		GROUP BY visitor_id
	`

	var profile types.VisitorProfileResponse
	row := a.clickDb.Db().QueryRow(ctx, summaryQuery, projectID, visitorID)
	if err := row.Scan(
		&profile.VisitorID,
		&profile.FirstSeen,
		&profile.LastSeen,
		&profile.TotalEvents,
		&profile.LocationCountryISO,
		&profile.LocationCity,
	); err != nil {
		return nil, fmt.Errorf("failed to get visitor summary: %w", err)
	}

	// Get first traffic origin (from the very first event)
	firstEventQuery := `
		SELECT
			referrer_domain,
			referrer_url
		FROM events
		WHERE project_id = ?
			AND visitor_id = ?
		ORDER BY client_timestamp_utc ASC
		LIMIT 1
	`

	firstEventRow := a.clickDb.Db().QueryRow(ctx, firstEventQuery, projectID, visitorID)
	if err := firstEventRow.Scan(&profile.FirstTrafficOrigin, &profile.FirstReferrerURL); err != nil {
		// It's okay if there's no first event, just continue
		profile.FirstTrafficOrigin = nil
		profile.FirstReferrerURL = nil
	}

	// Get visitor events
	eventsQuery := `
		SELECT
			event_name,
			client_timestamp_utc,
			page_url,
			page_path,
			referrer_url,
			device_type,
			browser_name
		FROM events
		WHERE project_id = ?
			AND visitor_id = ?
		ORDER BY client_timestamp_utc DESC
		LIMIT 100
	`

	eventsRows, err := a.clickDb.Db().Query(ctx, eventsQuery, projectID, visitorID)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitor events: %w", err)
	}
	defer eventsRows.Close()

	var events []types.VisitorEvent
	for eventsRows.Next() {
		var event types.VisitorEvent
		if err := eventsRows.Scan(
			&event.EventName,
			&event.ClientTimestampUTC,
			&event.PageURL,
			&event.PagePath,
			&event.ReferrerURL,
			&event.DeviceType,
			&event.BrowserName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}
		events = append(events, event)
	}
	profile.Events = events

	// Get events over time (last 30 days, grouped by day)
	now := time.Now().UTC()
	startTime := now.AddDate(0, 0, -30)

	eventsOverTimeQuery := `
		SELECT
			toStartOfDay(client_timestamp_utc) as time_bucket,
			count() as event_count
		FROM events
		WHERE project_id = ?
			AND visitor_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`

	timeRows, err := a.clickDb.Db().Query(ctx, eventsOverTimeQuery, projectID, visitorID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query events over time: %w", err)
	}
	defer timeRows.Close()

	var eventsOverTime []types.EventsOverTimeDataPoint
	for timeRows.Next() {
		var dp types.EventsOverTimeDataPoint
		if err := timeRows.Scan(&dp.Timestamp, &dp.EventCount); err != nil {
			return nil, fmt.Errorf("failed to scan events over time row: %w", err)
		}
		eventsOverTime = append(eventsOverTime, dp)
	}
	profile.EventsOverTime = eventsOverTime

	revenueQuery := `
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

	revenueRow := a.clickDb.Db().QueryRow(ctx, revenueQuery, projectID, visitorID)
	if err := revenueRow.Scan(
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
		LIMIT 20
	`

	paymentsRows, err := a.clickDb.Db().Query(ctx, paymentsQuery, projectID, visitorID)
	if err == nil {
		defer paymentsRows.Close()

		var payments []types.VisitorPayment
		for paymentsRows.Next() {
			var payment types.VisitorPayment
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

	// Fetch visitor identity data from PostgreSQL
	visitorIdentity, err := a.visitorRepository.GetVisitorByID(ctx, visitorID)
	if err != nil && err != sql.ErrNoRows {
		// Log error but don't fail the request - visitor might not be identified yet
		fmt.Printf("Warning: failed to fetch visitor identity: %v\n", err)
	}

	// Populate identity fields if visitor is identified
	if visitorIdentity != nil {
		profile.IsIdentified = true
		profile.UserID = visitorIdentity.UserID
		profile.ExternalID = visitorIdentity.ExternalID
		profile.Email = visitorIdentity.Email
		profile.Name = visitorIdentity.Name
		profile.Phone = visitorIdentity.Phone
		profile.CustomTraits = visitorIdentity.CustomTraits
		profile.FirstIdentifiedAt = visitorIdentity.FirstIdentifiedAt
		profile.LastIdentifiedAt = visitorIdentity.LastIdentifiedAt
	} else {
		profile.IsIdentified = false
	}

	return &profile, nil
}

// GetUniqueVisitorsTimeline returns unique visitor counts over time split by device type
func (a *AnalyticsData) GetUniqueVisitorsTimeline(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.UniqueVisitorsDataPoint, error) {
	startTime, intervalFunc, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT
			%s(client_timestamp_utc) as time_bucket,
			uniqIf(visitor_id, device_type = 'Mobile') as mobile,
			uniqIf(visitor_id, device_type = 'Desktop') as desktop
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`, intervalFunc)

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique visitors timeline: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.UniqueVisitorsDataPoint
	for rows.Next() {
		var dp types.UniqueVisitorsDataPoint
		if err := rows.Scan(&dp.Timestamp, &dp.Mobile, &dp.Desktop); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		dataPoints = append(dataPoints, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return dataPoints, nil
}

// GetSessionMetrics returns session metrics (duration, pages per session) for a time range
func (a *AnalyticsData) GetSessionMetrics(ctx context.Context, projectID string, timeRange types.TimeRange) (*types.SessionMetricsResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			COUNT(*) as total_sessions,
			AVG(dateDiff('second', started_at, ended_at)) as avg_duration_seconds,
			AVG(page_count) as avg_pages_per_session
		FROM sessions
		WHERE project_id = ?
			AND started_at >= ?
			AND started_at <= now()
	`

	var response types.SessionMetricsResponse
	row := a.clickDb.Db().QueryRow(ctx, query, projectID, startTime)
	if err := row.Scan(
		&response.TotalSessions,
		&response.AverageSessionDuration,
		&response.AveragePagesPerSession,
	); err != nil {
		return nil, fmt.Errorf("failed to query session metrics: %w", err)
	}

	return &response, nil
}

// GetBounceRate returns bounce rate metrics (overall and by page)
func (a *AnalyticsData) GetBounceRate(ctx context.Context, projectID string, timeRange types.TimeRange, limit int) (*types.BounceRateResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}

	// Overall bounce rate
	overallQuery := `
		SELECT
			countIf(page_count <= 1) * 100.0 / COUNT(*) as bounce_rate
		FROM sessions
		WHERE project_id = ?
			AND started_at >= ?
			AND started_at <= now()
	`

	var overallBounceRate float64
	row := a.clickDb.Db().QueryRow(ctx, overallQuery, projectID, startTime)
	if err := row.Scan(&overallBounceRate); err != nil {
		return nil, fmt.Errorf("failed to query overall bounce rate: %w", err)
	}

	// Bounce rate by page
	byPageQuery := `
		SELECT
			entry_page as page,
			COUNT(*) as sessions,
			countIf(page_count <= 1) * 100.0 / COUNT(*) as bounce_rate
		FROM sessions
		WHERE project_id = ?
			AND started_at >= ?
			AND started_at <= now()
		GROUP BY entry_page
		ORDER BY sessions DESC
		LIMIT ?
	`

	rows, err := a.clickDb.Db().Query(ctx, byPageQuery, projectID, startTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query bounce rate by page: %w", err)
	}
	defer rows.Close()

	var byPageMetrics []types.BounceRateByPageMetric
	for rows.Next() {
		var metric types.BounceRateByPageMetric
		if err := rows.Scan(&metric.Page, &metric.Sessions, &metric.BounceRate); err != nil {
			return nil, fmt.Errorf("failed to scan bounce rate by page: %w", err)
		}
		byPageMetrics = append(byPageMetrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bounce rate rows: %w", err)
	}

	return &types.BounceRateResponse{
		OverallBounceRate: overallBounceRate,
		ByPage:            byPageMetrics,
	}, nil
}

// GetActiveUsers returns DAU/WAU/MAU metrics
func (a *AnalyticsData) GetActiveUsers(ctx context.Context, projectID string) (*types.ActiveUsersResponse, error) {
	query := `
		SELECT
			countIf(last_seen >= now() - INTERVAL 1 DAY) as dau,
			countIf(last_seen >= now() - INTERVAL 7 DAY) as wau,
			countIf(last_seen >= now() - INTERVAL 30 DAY) as mau
		FROM visitor_summary
		WHERE project_id = ?
	`

	var response types.ActiveUsersResponse
	row := a.clickDb.Db().QueryRow(ctx, query, projectID)
	if err := row.Scan(&response.DAU, &response.WAU, &response.MAU); err != nil {
		return nil, fmt.Errorf("failed to query active users: %w", err)
	}

	return &response, nil
}

// GetReturnRate returns return rate metrics for a time range
func (a *AnalyticsData) GetReturnRate(ctx context.Context, projectID string, timeRange types.TimeRange) (*types.ReturnRateResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := `
		WITH user_stats AS (
			SELECT
				visitor_id,
				total_sessions as session_count,
				first_seen as first_seen,
				last_seen as last_seen
			FROM visitor_summary
			WHERE project_id = ?
				AND first_seen >= ?
		)
		SELECT
			COUNT(DISTINCT visitor_id) as total_users,
			COUNT(DISTINCT CASE WHEN session_count > 1 THEN visitor_id END) as returning_users,
			(COUNT(DISTINCT CASE WHEN session_count > 1 THEN visitor_id END) * 100.0 / COUNT(DISTINCT visitor_id)) as return_rate
		FROM user_stats
	`

	var response types.ReturnRateResponse
	row := a.clickDb.Db().QueryRow(ctx, query, projectID, startTime)
	if err := row.Scan(&response.TotalUsers, &response.ReturningUsers, &response.ReturnRatePercent); err != nil {
		return nil, fmt.Errorf("failed to query return rate: %w", err)
	}

	// Calculate average time between sessions for returning users
	timeBetweenQuery := `
		WITH session_times AS (
			SELECT
				visitor_id,
				started_at as session_start,
				ROW_NUMBER() OVER (PARTITION BY visitor_id ORDER BY started_at) as session_num
			FROM sessions
			WHERE project_id = ?
				AND started_at >= ?
		),
		session_gaps AS (
			SELECT
				s1.visitor_id,
				dateDiff('hour', s1.session_start, s2.session_start) as hours_between
			FROM session_times s1
			JOIN session_times s2
				ON s1.visitor_id = s2.visitor_id
				AND s2.session_num = s1.session_num + 1
		)
		SELECT AVG(hours_between) as avg_hours
		FROM session_gaps
	`

	var avgTimeBetween float64
	avgRow := a.clickDb.Db().QueryRow(ctx, timeBetweenQuery, projectID, startTime)
	if err := avgRow.Scan(&avgTimeBetween); err == nil {
		response.AvgTimeBetweenSessions = avgTimeBetween
	}

	return &response, nil
}

// GetChurnRate returns churn rate metrics
func (a *AnalyticsData) GetChurnRate(ctx context.Context, projectID string, timeRange types.TimeRange, churnThresholdDays int) (*types.ChurnRateResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	if churnThresholdDays <= 0 {
		churnThresholdDays = 30
	}

	query := fmt.Sprintf(`
		WITH user_stats AS (
			SELECT
				visitor_id,
				first_seen as first_seen,
				last_seen as last_seen
			FROM visitor_summary
			WHERE project_id = ?
				AND first_seen >= ?
		)
		SELECT
			COUNT(DISTINCT visitor_id) as total_users,
			COUNT(DISTINCT CASE
				WHEN last_seen < now() - INTERVAL %d DAY
				THEN visitor_id
			END) as churned_users,
			(COUNT(DISTINCT CASE
				WHEN last_seen < now() - INTERVAL %d DAY
				THEN visitor_id
			END) * 100.0 / COUNT(DISTINCT visitor_id)) as churn_rate
		FROM user_stats
	`, churnThresholdDays, churnThresholdDays)

	var response types.ChurnRateResponse
	response.ChurnThresholdDays = churnThresholdDays

	row := a.clickDb.Db().QueryRow(ctx, query, projectID, startTime)
	if err := row.Scan(&response.TotalUsers, &response.ChurnedUsers, &response.ChurnRatePercent); err != nil {
		return nil, fmt.Errorf("failed to query churn rate: %w", err)
	}

	return &response, nil
}

// GetCohortAnalysis returns cohort retention analysis
func (a *AnalyticsData) GetCohortAnalysis(ctx context.Context, projectID string, timeRange types.TimeRange) (*types.CohortAnalysisResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := `
		WITH user_cohorts AS (
			SELECT
				visitor_id,
				toStartOfWeek(first_seen) as cohort_week,
				first_seen as first_seen,
				last_seen as last_seen
			FROM visitor_summary
			WHERE project_id = ?
				AND first_seen >= ?
		)
		SELECT
			cohort_week,
			COUNT(DISTINCT visitor_id) as cohort_size,
			COUNT(DISTINCT CASE
				WHEN last_seen >= first_seen + INTERVAL 7 DAY
				THEN visitor_id
			END) * 100.0 / COUNT(DISTINCT visitor_id) as week_1_retention,
			COUNT(DISTINCT CASE
				WHEN last_seen >= first_seen + INTERVAL 14 DAY
				THEN visitor_id
			END) * 100.0 / COUNT(DISTINCT visitor_id) as week_2_retention,
			COUNT(DISTINCT CASE
				WHEN last_seen >= first_seen + INTERVAL 28 DAY
				THEN visitor_id
			END) * 100.0 / COUNT(DISTINCT visitor_id) as week_4_retention,
			COUNT(DISTINCT CASE
				WHEN last_seen >= first_seen + INTERVAL 30 DAY
				THEN visitor_id
			END) * 100.0 / COUNT(DISTINCT visitor_id) as month_1_retention,
			COUNT(DISTINCT CASE
				WHEN last_seen >= first_seen + INTERVAL 60 DAY
				THEN visitor_id
			END) * 100.0 / COUNT(DISTINCT visitor_id) as month_2_retention,
			COUNT(DISTINCT CASE
				WHEN last_seen >= first_seen + INTERVAL 90 DAY
				THEN visitor_id
			END) * 100.0 / COUNT(DISTINCT visitor_id) as month_3_retention
		FROM user_cohorts
		GROUP BY cohort_week
		ORDER BY cohort_week DESC
		LIMIT 20
	`

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query cohort analysis: %w", err)
	}
	defer rows.Close()

	var cohorts []types.CohortData
	for rows.Next() {
		var cohort types.CohortData
		if err := rows.Scan(
			&cohort.CohortPeriod,
			&cohort.CohortSize,
			&cohort.Week1Retention,
			&cohort.Week2Retention,
			&cohort.Week4Retention,
			&cohort.Month1Retention,
			&cohort.Month2Retention,
			&cohort.Month3Retention,
		); err != nil {
			return nil, fmt.Errorf("failed to scan cohort data: %w", err)
		}
		cohorts = append(cohorts, cohort)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cohort rows: %w", err)
	}

	return &types.CohortAnalysisResponse{
		Cohorts: cohorts,
	}, nil
}

// GetTotalRevenue returns total revenue for all payments within a time period
func (a *AnalyticsData) GetTotalRevenue(ctx context.Context, projectID string, startTime time.Time) (int64, string, error) {
	query := `
		SELECT
			COALESCE(SUM(amount), 0) as total_revenue,
			any(currency) as currency
		FROM payment_events
		WHERE project_id = ?
			AND payment_timestamp_utc >= ?
			AND payment_timestamp_utc <= now()
			AND payment_status = 'succeeded'
	`

	var totalRevenue int64
	var currency string
	row := a.clickDb.Db().QueryRow(ctx, query, projectID, startTime)
	if err := row.Scan(&totalRevenue, &currency); err != nil {
		return 0, "USD", fmt.Errorf("failed to get total revenue: %w", err)
	}

	if currency == "" {
		currency = "USD"
	}

	return totalRevenue, currency, nil
}

// GetIdentifiedCustomersRevenue returns total revenue for identified customers only
func (a *AnalyticsData) GetIdentifiedCustomersRevenue(ctx context.Context, projectID string, startTime time.Time) (int64, error) {
	query := `
		SELECT
			COALESCE(SUM(p.amount), 0) as total_revenue
		FROM payment_events p
		WHERE p.project_id = ?
			AND p.payment_timestamp_utc >= ?
			AND p.payment_timestamp_utc <= now()
			AND p.payment_status = 'succeeded'
			AND p.visitor_id IN (
				SELECT DISTINCT visitor_id
				FROM events
				WHERE project_id = ?
					AND (user_id IS NOT NULL OR external_id IS NOT NULL OR email_hash IS NOT NULL)
			)
	`

	var totalRevenue int64
	row := a.clickDb.Db().QueryRow(ctx, query, projectID, startTime, projectID)
	if err := row.Scan(&totalRevenue); err != nil {
		return 0, fmt.Errorf("failed to get identified customers revenue: %w", err)
	}

	return totalRevenue, nil
}

func (a *AnalyticsData) GetAvgRevenuePerSession(ctx context.Context, projectID string, startTime time.Time) (int64, error) {

	revenueQuery := `
		SELECT COALESCE(SUM(amount), 0) as total_revenue
		FROM payment_events
		WHERE project_id = ?
			AND payment_timestamp_utc >= ?
			AND payment_timestamp_utc <= now()
			AND payment_status = 'succeeded'
	`

	var totalRevenue int64
	revenueRow := a.clickDb.Db().QueryRow(ctx, revenueQuery, projectID, startTime)
	if err := revenueRow.Scan(&totalRevenue); err != nil {
		return 0, fmt.Errorf("failed to get total revenue: %w", err)
	}

	sessionsQuery := `
		SELECT COUNT(DISTINCT session_id) as unique_sessions
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
	`

	var uniqueSessions uint64
	sessionsRow := a.clickDb.Db().QueryRow(ctx, sessionsQuery, projectID, startTime)
	if err := sessionsRow.Scan(&uniqueSessions); err != nil {
		return 0, fmt.Errorf("failed to get unique sessions: %w", err)
	}

	if uniqueSessions == 0 {
		return 0, nil
	}

	return int64(float64(totalRevenue) / float64(uniqueSessions)), nil
}

// GetConversionRate returns percentage of unique visitors who made a payment
func (a *AnalyticsData) GetConversionRate(ctx context.Context, projectID string, startTime time.Time) (float64, uint64, uint64, error) {
	// Get unique visitors count
	visitorsQuery := `
		SELECT COUNT(DISTINCT visitor_id) as unique_visitors
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
	`

	var uniqueVisitors uint64
	visitorsRow := a.clickDb.Db().QueryRow(ctx, visitorsQuery, projectID, startTime)
	if err := visitorsRow.Scan(&uniqueVisitors); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get unique visitors: %w", err)
	}

	// Get paying visitors count
	payingQuery := `
		SELECT COUNT(DISTINCT visitor_id) as paying_visitors
		FROM payment_events
		WHERE project_id = ?
			AND payment_timestamp_utc >= ?
			AND payment_timestamp_utc <= now()
			AND payment_status = 'succeeded'
	`

	var payingVisitors uint64
	payingRow := a.clickDb.Db().QueryRow(ctx, payingQuery, projectID, startTime)
	if err := payingRow.Scan(&payingVisitors); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get paying visitors: %w", err)
	}

	if uniqueVisitors == 0 {
		return 0, uniqueVisitors, payingVisitors, nil
	}

	conversionRate := float64(payingVisitors) * 100.0 / float64(uniqueVisitors)
	return conversionRate, uniqueVisitors, payingVisitors, nil
}

// GetDashboardMetrics returns combined key metrics for a dashboard using parallel queries
func (a *AnalyticsData) GetDashboardMetrics(ctx context.Context, projectID string, timeRange types.TimeRange) (*types.DashboardMetricsResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	// Use errgroup to run all queries in parallel
	type results struct {
		// Existing metrics
		activeUsers                     *types.ActiveUsersResponse
		sessionMetrics                  *types.SessionMetricsResponse
		bounceRate                      *types.BounceRateResponse
		returnRate                      *types.ReturnRateResponse
		sessionsToday                   uint64
		totalEvents                     uint64
		uniqueVisitors                  uint64
		uniqueSessions                  uint64
		payingVisitors                  uint64
		totalPayments                   uint64
		avgRevenuePerVisitor            float64
		avgRevenuePerIdentifiedCustomer float64

		totalRevenue                    int64
		totalRevenueIdentifiedCustomers int64
		avgRevenuePerSession            int64
		conversionRate                  float64

		currency string
	}

	var res results
	var g errgroup.Group

	// 1. Get active users
	g.Go(func() error {
		activeUsers, err := a.GetActiveUsers(ctx, projectID)
		if err != nil {
			return fmt.Errorf("failed to get active users: %w", err)
		}
		res.activeUsers = activeUsers
		return nil
	})

	// 2. Get session metrics
	g.Go(func() error {
		sessionMetrics, err := a.GetSessionMetrics(ctx, projectID, timeRange)
		if err != nil {
			return fmt.Errorf("failed to get session metrics: %w", err)
		}
		res.sessionMetrics = sessionMetrics
		return nil
	})

	// 3. Get bounce rate
	g.Go(func() error {
		bounceRate, err := a.GetBounceRate(ctx, projectID, timeRange, 0)
		if err != nil {
			return fmt.Errorf("failed to get bounce rate: %w", err)
		}
		res.bounceRate = bounceRate
		return nil
	})

	// 4. Get return rate
	g.Go(func() error {
		returnRate, err := a.GetReturnRate(ctx, projectID, timeRange)
		if err != nil {
			return fmt.Errorf("failed to get return rate: %w", err)
		}
		res.returnRate = returnRate
		return nil
	})

	// 5. Get sessions today
	g.Go(func() error {
		now := time.Now().UTC()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		query := `
			SELECT COUNT(*) as sessions_today
			FROM sessions
			WHERE project_id = ?
				AND started_at >= ?
		`
		row := a.clickDb.Db().QueryRow(ctx, query, projectID, todayStart)
		if err := row.Scan(&res.sessionsToday); err != nil {
			return fmt.Errorf("failed to get sessions today: %w", err)
		}
		return nil
	})

	// 6. Get total events, unique visitors, and unique sessions
	g.Go(func() error {
		query := `
			SELECT
				COUNT(*) as total_events,
				uniq(visitor_id) as unique_visitors,
				uniq(session_id) as unique_sessions
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
		`
		row := a.clickDb.Db().QueryRow(ctx, query, projectID, startTime)
		if err := row.Scan(&res.totalEvents, &res.uniqueVisitors, &res.uniqueSessions); err != nil {
			return fmt.Errorf("failed to get totals: %w", err)
		}
		return nil
	})

	// 7. Get total revenue for all payments (NEW METRIC #1)
	g.Go(func() error {
		totalRevenue, currency, err := a.GetTotalRevenue(ctx, projectID, startTime)
		if err != nil {
			return err
		}
		res.totalRevenue = totalRevenue
		res.currency = currency
		return nil
	})

	// 8. Get total revenue for identified customers only (NEW METRIC #2)
	g.Go(func() error {
		totalRevenueIdentified, err := a.GetIdentifiedCustomersRevenue(ctx, projectID, startTime)
		if err != nil {
			return err
		}
		res.totalRevenueIdentifiedCustomers = totalRevenueIdentified
		return nil
	})

	// 9. Get average revenue per session (NEW METRIC #3)
	g.Go(func() error {
		avgRevPerSession, err := a.GetAvgRevenuePerSession(ctx, projectID, startTime)
		if err != nil {
			return err
		}
		res.avgRevenuePerSession = avgRevPerSession
		return nil
	})

	// 10. Get conversion rate (NEW METRIC #4)
	g.Go(func() error {
		convRate, _, payingVis, err := a.GetConversionRate(ctx, projectID, startTime)
		if err != nil {
			return err
		}
		res.conversionRate = convRate
		res.payingVisitors = payingVis
		return nil
	})

	// 11. Get total payments and avg revenue per paying visitor (legacy metrics)
	g.Go(func() error {
		query := `
			SELECT
				COUNT(*) as total_payments,
				CASE
					WHEN uniq(visitor_id) > 0
					THEN COALESCE(SUM(amount), 0) / uniq(visitor_id)
					ELSE 0
				END as avg_revenue_per_visitor
			FROM payment_events
			WHERE project_id = ?
				AND payment_timestamp_utc >= ?
				AND payment_timestamp_utc <= now()
				AND payment_status = 'succeeded'
		`
		row := a.clickDb.Db().QueryRow(ctx, query, projectID, startTime)
		if err := row.Scan(&res.totalPayments, &res.avgRevenuePerVisitor); err != nil {
			// Set defaults if no payments found
			res.totalPayments = 0
			res.avgRevenuePerVisitor = 0
		}
		return nil
	})

	// 12. Get average revenue per identified customer (legacy metric)
	g.Go(func() error {
		query := `
			SELECT
				CASE
					WHEN uniq(p.visitor_id) > 0
					THEN COALESCE(SUM(p.amount), 0) / uniq(p.visitor_id)
					ELSE 0
				END as avg_revenue_identified
			FROM payment_events p
			WHERE p.project_id = ?
				AND p.payment_timestamp_utc >= ?
				AND p.payment_timestamp_utc <= now()
				AND p.payment_status = 'succeeded'
				AND p.visitor_id IN (
					SELECT DISTINCT visitor_id
					FROM events
					WHERE project_id = ?
						AND (user_id IS NOT NULL OR external_id IS NOT NULL OR email_hash IS NOT NULL)
				)
		`
		row := a.clickDb.Db().QueryRow(ctx, query, projectID, startTime, projectID)
		if err := row.Scan(&res.avgRevenuePerIdentifiedCustomer); err != nil {
			res.avgRevenuePerIdentifiedCustomer = 0
		}
		return nil
	})

	// Wait for all queries to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &types.DashboardMetricsResponse{
		// Active users
		DAU: res.activeUsers.DAU,
		WAU: res.activeUsers.WAU,
		MAU: res.activeUsers.MAU,

		// Sessions
		SessionsToday:         res.sessionsToday,
		TotalSessionsInPeriod: res.sessionMetrics.TotalSessions,
		AvgSessionDuration:    res.sessionMetrics.AverageSessionDuration,
		AvgPagesPerSession:    res.sessionMetrics.AveragePagesPerSession,

		// Engagement metrics
		BounceRate: res.bounceRate.OverallBounceRate,
		ReturnRate: res.returnRate.ReturnRatePercent,

		// Total metrics
		TotalEvents:    res.totalEvents,
		UniqueVisitors: res.uniqueVisitors,
		UniqueSessions: res.uniqueSessions,

		// 4 new revenue metrics
		TotalRevenue:                    res.totalRevenue,
		TotalRevenueIdentifiedCustomers: res.totalRevenueIdentifiedCustomers,
		AvgRevenuePerSession:            res.avgRevenuePerSession,
		ConversionRate:                  res.conversionRate,

		// Legacy revenue metrics (kept for compatibility)
		PayingVisitors:                  res.payingVisitors,
		ConversionToPaying:              res.conversionRate, // Same as ConversionRate
		AvgRevenuePerVisitor:            res.avgRevenuePerVisitor,
		AvgRevenuePerIdentifiedCustomer: res.avgRevenuePerIdentifiedCustomer,
		TotalPayments:                   res.totalPayments,
		Currency:                        res.currency,
	}, nil
}

// GetRevenueByUTMSource returns revenue grouped by UTM parameters using first-touch attribution
func (a *AnalyticsData) GetRevenueByUTMSource(ctx context.Context, projectID string, timeRange types.TimeRange, utmType string) ([]types.UTMRevenueDataPoint, error) {
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

	// Use visitor_first_touch_attribution table for cached first-touch attribution
	// This shows ALL payments made in the time period, attributed to the visitor's original UTM source
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
			uniq(p.visitor_id) as paying_visitors,
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
			END as avg_revenue_per_visitor,
			countIf(p.payment_id IS NOT NULL) as payment_count,
			any(p.currency) as currency
		FROM visitor_utm vu
		LEFT JOIN payments_in_period p ON vu.visitor_id = p.visitor_id
		GROUP BY vu.utm_value
		ORDER BY total_revenue DESC
		LIMIT 20
	`, utmField, utmField, utmField, utmField, utmField, utmField)

	rows, err := a.clickDb.Db().Query(ctx, query,
		projectID, startTime, // revenue_totals CTE
		projectID, startTime, // payments_in_period CTE
		projectID,            // visitor_utm CTE
		projectID, startTime, // visitors_in_period CTE
		projectID, // all_visitor_utm CTE
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue by UTM: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.UTMRevenueDataPoint
	for rows.Next() {
		var dp types.UTMRevenueDataPoint
		if err := rows.Scan(
			&dp.UTMValue,
			&dp.TotalRevenue,
			&dp.RevenuePercentage,
			&dp.PayingVisitors,
			&dp.UniqueVisitors,
			&dp.ConversionRate,
			&dp.AvgRevenuePerVisitor,
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

// GetRevenueTimeline returns revenue over time using materialized views for performance
func (a *AnalyticsData) GetRevenueTimeline(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.RevenueTimelineDataPoint, error) {
	startTime, intervalFunc, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	// Use materialized views for better performance
	// Choose the appropriate view based on the time range granularity
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
		// Fall back to raw query for sub-hour granularity (last_hour)
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

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue timeline: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.RevenueTimelineDataPoint
	for rows.Next() {
		var dp types.RevenueTimelineDataPoint
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
