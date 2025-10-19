package data

import (
	"context"
	"fmt"
	"time"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/types"
)

type AnalyticsData struct {
	clickDb *clickhouse.ClickhouseDB
}

func NewAnalyticsData(clickDb *clickhouse.ClickhouseDB) *AnalyticsData {
	return &AnalyticsData{clickDb: clickDb}
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
			countIf(device_type = 'mobile') as mobile,
			countIf(device_type = 'desktop') as desktop,
			countIf(device_type = 'tablet') as tablet,
			countIf(device_type IS NULL OR (device_type != 'mobile' AND device_type != 'desktop' AND device_type != 'tablet')) as unknown
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
		if err := rows.Scan(&dp.Timestamp, &dp.Mobile, &dp.Desktop, &dp.Tablet, &dp.Unknown); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		dataPoints = append(dataPoints, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return dataPoints, nil
}

// GetUniqueVisitorsByOrigin returns unique visitor counts grouped by traffic origin
func (a *AnalyticsData) GetUniqueVisitorsByOrigin(ctx context.Context, projectID string, timeRange types.TimeRange) ([]types.OriginDataPoint, error) {
	startTime, _, err := GetTimeRangeBounds(timeRange)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			CASE
				WHEN referrer_domain IS NULL OR referrer_domain = '' THEN 'Direct'
				ELSE referrer_domain
			END as origin,
			uniq(visitor_id) as unique_visitors
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY origin
		ORDER BY unique_visitors DESC
		LIMIT 20
	`

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitors by origin: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.OriginDataPoint
	for rows.Next() {
		var dp types.OriginDataPoint
		if err := rows.Scan(&dp.Origin, &dp.UniqueVisitors); err != nil {
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
		SELECT
			CASE
				WHEN location_country_iso IS NULL OR location_country_iso = '' THEN 'Unknown'
				ELSE location_country_iso
			END as country_code,
			uniq(visitor_id) as unique_visitors
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY country_code
		ORDER BY unique_visitors DESC
		LIMIT 50
	`

	rows, err := a.clickDb.Db().Query(ctx, query, projectID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitors by country: %w", err)
	}
	defer rows.Close()

	var dataPoints []types.CountryDataPoint
	for rows.Next() {
		var dp types.CountryDataPoint
		if err := rows.Scan(&dp.CountryCode, &dp.UniqueVisitors); err != nil {
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

// GetTopVisitors returns the most active visitors for a project within a time range
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
			uniqIf(visitor_id, device_type = 'mobile') as mobile,
			uniqIf(visitor_id, device_type = 'desktop') as desktop
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
