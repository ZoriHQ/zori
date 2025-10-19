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
