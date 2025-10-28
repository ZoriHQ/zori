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

// GetUniqueVisitorsByOrigin returns unique visitor counts grouped by traffic origin using first-touch attribution
func (a *AnalyticsData) GetUniqueVisitorsByOrigin(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.OriginDataPoint, error) {
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
		)
		SELECT
			vo.origin,
			uniq(vo.visitor_id) as unique_visitors,
			uniq(vo.visitor_id) * 100.0 / (SELECT COUNT(DISTINCT visitor_id) FROM visitors_in_period) as percentage
		FROM visitor_origins vo
		GROUP BY vo.origin
		ORDER BY unique_visitors DESC
		LIMIT 20
	`

	rows, err := a.clickDb.Db().Query(ctx, query,
		projectID, startTime, // visitors_in_period CTE
		projectID,            // visitor_origins CTE
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

// GetRecentEvents returns the most recent events for a project with optional filters
func (a *AnalyticsData) GetRecentEvents(ctx context.Context, req types.RecentEventsRequest) ([]types.RecentEvent, uint64, error) {
	if req.Limit <= 0 {
		req.Limit = 15
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Build WHERE clause dynamically based on filters
	// Add default time range to prevent counting entire table (performance optimization)
	now := time.Now().UTC()
	defaultStartTime := now.AddDate(0, 0, -7) // Last 7 days by default

	whereConditions := []string{"project_id = ?", "client_timestamp_utc >= ?"}
	args := []interface{}{req.ProjectID, defaultStartTime}

	if req.VisitorID != nil && *req.VisitorID != "" {
		whereConditions = append(whereConditions, "visitor_id = ?")
		args = append(args, *req.VisitorID)
	}

	if req.UserID != nil && *req.UserID != "" {
		whereConditions = append(whereConditions, "user_id = ?")
		args = append(args, *req.UserID)
	}

	if req.ExternalID != nil && *req.ExternalID != "" {
		whereConditions = append(whereConditions, "external_id = ?")
		args = append(args, *req.ExternalID)
	}

	if req.TrafficOrigin != nil && *req.TrafficOrigin != "" {
		whereConditions = append(whereConditions, "referrer_domain = ?")
		args = append(args, *req.TrafficOrigin)
	}

	if req.PagePath != nil && *req.PagePath != "" {
		whereConditions = append(whereConditions, "page_path = ?")
		args = append(args, *req.PagePath)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + whereConditions[i]
		}
	}

	// Get approximate total count for pagination using count() which is faster than COUNT(*)
	// ClickHouse's count() is optimized and uses metadata when possible
	countQuery := fmt.Sprintf(`
		SELECT count()
		FROM events
		%s
	`, whereClause)

	var totalCount uint64
	countRow := a.clickDb.Db().QueryRow(ctx, countQuery, args...)
	if err := countRow.Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Build main query with all fields including lat/long and click data
	query := fmt.Sprintf(`
		SELECT
			event_name,
			visitor_id,
			user_id,
			external_id,
			client_timestamp_utc,
			page_url,
			page_path,
			referrer_url,
			referrer_domain,
			device_type,
			browser_name,
			location_country_iso,
			location_city,
			location_latitude,
			location_longitude,
			click_element_tag,
			click_element_selector,
			click_element_text,
			click_position_x,
			click_position_y,
			click_screen_width,
			click_screen_height,
			click_element_type,
			click_element_category,
			is_cta_click,
			link_destination,
			is_external_link,
			is_download_link
		FROM events
		%s
		ORDER BY client_timestamp_utc DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	// Create a new slice with limit and offset appended (don't modify original args)
	queryArgs := make([]interface{}, len(args), len(args)+2)
	copy(queryArgs, args)
	queryArgs = append(queryArgs, req.Limit, req.Offset)

	rows, err := a.clickDb.Db().Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query recent events: %w", err)
	}
	defer rows.Close()

	var events []types.RecentEvent
	for rows.Next() {
		var event types.RecentEvent
		if err := rows.Scan(
			&event.EventName,
			&event.VisitorID,
			&event.UserID,
			&event.ExternalID,
			&event.ClientTimestampUTC,
			&event.PageURL,
			&event.PagePath,
			&event.ReferrerURL,
			&event.ReferrerDomain,
			&event.DeviceType,
			&event.BrowserName,
			&event.LocationCountryISO,
			&event.LocationCity,
			&event.LocationLatitude,
			&event.LocationLongitude,
			&event.ClickElementTag,
			&event.ClickElementSelector,
			&event.ClickElementText,
			&event.ClickPositionX,
			&event.ClickPositionY,
			&event.ClickScreenWidth,
			&event.ClickScreenHeight,
			&event.ClickElementType,
			&event.ClickElementCategory,
			&event.IsCTAClick,
			&event.LinkDestination,
			&event.IsExternalLink,
			&event.IsDownloadLink,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	return events, totalCount, nil
}

// GetEventFilterOptions returns unique traffic origins and page paths for filter dropdowns
func (a *AnalyticsData) GetEventFilterOptions(ctx context.Context, projectID string, timeRange *types.TimeRange) (*types.EventFilterOptionsResponse, error) {
	// Build WHERE clause based on optional time range
	whereClause := "WHERE project_id = ?"
	args := []interface{}{projectID}

	if timeRange != nil && *timeRange != "" {
		startTime, _, err := GetTimeRangeBounds(*timeRange)
		if err == nil {
			whereClause += " AND client_timestamp_utc >= ? AND client_timestamp_utc <= now()"
			args = append(args, startTime)
		}
	}

	// Query for unique traffic origins (referrer domains)
	originsQuery := fmt.Sprintf(`
		SELECT DISTINCT referrer_domain
		FROM events
		%s
			AND referrer_domain IS NOT NULL
			AND referrer_domain != ''
		ORDER BY referrer_domain
		LIMIT 1000
	`, whereClause)

	originsRows, err := a.clickDb.Db().Query(ctx, originsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query traffic origins: %w", err)
	}
	defer originsRows.Close()

	var origins []string
	for originsRows.Next() {
		var origin string
		if err := originsRows.Scan(&origin); err != nil {
			return nil, fmt.Errorf("failed to scan origin: %w", err)
		}
		origins = append(origins, origin)
	}

	if err := originsRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating origins: %w", err)
	}

	// Query for unique page paths
	pagesQuery := fmt.Sprintf(`
		SELECT DISTINCT page_path
		FROM events
		%s
			AND page_path IS NOT NULL
			AND page_path != ''
		ORDER BY page_path
		LIMIT 1000
	`, whereClause)

	pagesRows, err := a.clickDb.Db().Query(ctx, pagesQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pages: %w", err)
	}
	defer pagesRows.Close()

	var pages []string
	for pagesRows.Next() {
		var page string
		if err := pagesRows.Scan(&page); err != nil {
			return nil, fmt.Errorf("failed to scan page: %w", err)
		}
		pages = append(pages, page)
	}

	if err := pagesRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pages: %w", err)
	}

	return &types.EventFilterOptionsResponse{
		TrafficOrigins: origins,
		Pages:          pages,
	}, nil
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
			visitor_id,
			count() as event_count,
			max(client_timestamp_utc) as last_seen,
			min(client_timestamp_utc) as first_seen,
			any(location_country_iso) as location_country_iso,
			any(location_city) as location_city,
			any(device_type) as device_type,
			any(browser_name) as browser_name
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY visitor_id
		ORDER BY event_count DESC
		LIMIT ?
	`

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime, limit)
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
			browser_name,
			click_element_tag,
			click_element_selector,
			click_element_text,
			click_position_x,
			click_position_y,
			click_screen_width,
			click_screen_height,
			click_element_type,
			click_element_category,
			is_cta_click,
			link_destination,
			is_external_link,
			is_download_link
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
			&event.ClickElementTag,
			&event.ClickElementSelector,
			&event.ClickElementText,
			&event.ClickPositionX,
			&event.ClickPositionY,
			&event.ClickScreenWidth,
			&event.ClickScreenHeight,
			&event.ClickElementType,
			&event.ClickElementCategory,
			&event.IsCTAClick,
			&event.LinkDestination,
			&event.IsExternalLink,
			&event.IsDownloadLink,
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

// GetDashboardMetrics returns combined key metrics for a dashboard using parallel queries
func (a *AnalyticsData) GetDashboardMetrics(ctx context.Context, projectID string, timeRange types.TimeRange) (*types.DashboardMetricsResponse, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	// Use errgroup to run all queries in parallel
	type results struct {
		activeUsers    *types.ActiveUsersResponse
		sessionMetrics *types.SessionMetricsResponse
		bounceRate     *types.BounceRateResponse
		returnRate     *types.ReturnRateResponse
		sessionsToday  uint64
		totalEvents    uint64
		uniqueVisitors uint64
		uniqueSessions uint64
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
	}, nil
}
