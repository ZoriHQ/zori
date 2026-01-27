package data

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
	"zori/services/analytics/types"
)

type AnalyticsData struct {
	clickDb *clickhouse.ClickhouseDB
}

func NewAnalyticsData(clickDb *clickhouse.ClickhouseDB) *AnalyticsData {
	return &AnalyticsData{
		clickDb: clickDb,
	}
}

func (a *AnalyticsData) GetVisitorsByDevice(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]types.VisitorDataPoint, error) {
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
	`, filter.TimeRange.TimeBucketFunction)

	rows, err := a.clickDb.Db().Query(ctx, query, filter.ProjectID, filter.TimeRange.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitors by device: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

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

	if dataPoints == nil {
		dataPoints = []types.VisitorDataPoint{}
	}

	return dataPoints, nil
}

func (a *AnalyticsData) GetRecentEvents(ctx *ctx.Ctx, req *types.RecentEventsRequest) ([]types.RecentEvent, uint64, error) {
	now := time.Now().UTC()
	defaultStartTime := now.AddDate(0, 0, -7)

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

	if req.EventName != nil && *req.EventName != "" {
		eventNames := strings.Split(*req.EventName, ",")
		// Trim whitespace from each event name
		trimmedEventNames := make([]string, 0, len(eventNames))
		for _, name := range eventNames {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				trimmedEventNames = append(trimmedEventNames, trimmed)
			}
		}
		if len(trimmedEventNames) > 0 {
			// Build IN clause with placeholders
			placeholders := make([]string, len(trimmedEventNames))
			for i := range trimmedEventNames {
				placeholders[i] = "?"
			}
			whereConditions = append(whereConditions, fmt.Sprintf("event_name IN (%s)", strings.Join(placeholders, ",")))
			// Add each event name as a separate argument
			for _, name := range trimmedEventNames {
				args = append(args, name)
			}
		}
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + whereConditions[i]
		}
	}

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

	query := fmt.Sprintf(`
		SELECT
			event_name,
			visitor_id,
			session_id,
			user_id,
			external_id,
			client_timestamp_utc,
			page_url,
			page_path,
			host,
			referrer_url,
			referrer_domain,
			device_type,
			browser_name,
			os_name,
			location_country_iso,
			location_city,
			location_latitude,
			location_longitude,
			utm_source,
			utm_medium,
			utm_campaign,
			custom_properties,
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

	queryArgs := make([]interface{}, len(args), len(args)+2)
	copy(queryArgs, args)
	queryArgs = append(queryArgs, req.Limit, req.Offset)

	rows, err := a.clickDb.Db().Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query recent events: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

	var events []types.RecentEvent
	for rows.Next() {
		var event types.RecentEvent
		var customPropertiesStr string
		var utmSource, utmMedium, utmCampaign string
		if err := rows.Scan(
			&event.EventName,
			&event.VisitorID,
			&event.SessionID,
			&event.UserID,
			&event.ExternalID,
			&event.ClientTimestampUTC,
			&event.PageURL,
			&event.PagePath,
			&event.Host,
			&event.ReferrerURL,
			&event.ReferrerDomain,
			&event.DeviceType,
			&event.BrowserName,
			&event.OsName,
			&event.LocationCountryISO,
			&event.LocationCity,
			&event.LocationLatitude,
			&event.LocationLongitude,
			&utmSource,
			&utmMedium,
			&utmCampaign,
			&customPropertiesStr,
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

		// Set UTM parameters only if they have values
		if utmSource != "" {
			event.UTMSource = &utmSource
		}
		if utmMedium != "" {
			event.UTMMedium = &utmMedium
		}
		if utmCampaign != "" {
			event.UTMCampaign = &utmCampaign
		}

		// Parse custom properties JSON string into map
		if customPropertiesStr != "" && customPropertiesStr != "{}" {
			var customProps map[string]any
			if err := json.Unmarshal([]byte(customPropertiesStr), &customProps); err == nil && len(customProps) > 0 {
				event.CustomProperties = customProps
			}
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	return events, totalCount, nil
}

func (a *AnalyticsData) GetEventFilterOptions(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.EventFilterOptionsResponse, error) {
	whereClause := "WHERE project_id = ?"
	args := []any{filter.ProjectID}

	if filter.TimeRange != nil {
		whereClause += " AND client_timestamp_utc >= ? AND client_timestamp_utc <= now()"
		args = append(args, filter.TimeRange.Start)
	}

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
	defer clickhouse.EnsureClosed(originsRows)

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
	defer clickhouse.EnsureClosed(pagesRows)

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

	eventNamesQuery := `
		SELECT event_name
		FROM unique_event_names
		WHERE project_id = ?
		ORDER BY event_name
		LIMIT 1000
	`

	eventNamesRows, err := a.clickDb.Db().Query(ctx, eventNamesQuery, filter.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query event names: %w", err)
	}
	defer clickhouse.EnsureClosed(eventNamesRows)

	var eventNames []string
	for eventNamesRows.Next() {
		var eventName string
		if err := eventNamesRows.Scan(&eventName); err != nil {
			return nil, fmt.Errorf("failed to scan event name: %w", err)
		}
		eventNames = append(eventNames, eventName)
	}

	if err := eventNamesRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event names: %w", err)
	}

	if origins == nil {
		origins = []string{}
	}
	if pages == nil {
		pages = []string{}
	}
	if eventNames == nil {
		eventNames = []string{}
	}

	return &types.EventFilterOptionsResponse{
		TrafficOrigins: origins,
		Pages:          pages,
		EventNames:     eventNames,
	}, nil
}

func (a *AnalyticsData) GetTopVisitors(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]types.TopVisitor, error) {
	// Get visitor data with identity information directly from events table
	visitorDataQuery := `
		SELECT
			visitor_id,
			count() as event_count,
			min(client_timestamp_utc) as first_seen,
			max(client_timestamp_utc) as last_seen,
			any(location_country_iso) as location_country_iso,
			any(location_city) as location_city,
			any(device_type) as device_type,
			any(browser_name) as browser_name,
			anyIf(user_id, user_id IS NOT NULL AND user_id != '') as user_id,
			anyIf(external_id, external_id IS NOT NULL AND external_id != '') as external_id
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY visitor_id
		ORDER BY event_count DESC
	`

	rows, err := a.clickDb.Db().Query(ctx, visitorDataQuery, filter.ProjectID, filter.TimeRange.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitor data: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

	type visitorData struct {
		VisitorID          string
		EventCount         uint64
		FirstSeen          time.Time
		LastSeen           time.Time
		LocationCountryISO *string
		LocationCity       *string
		DeviceType         *string
		BrowserName        *string
		UserID             *string
		ExternalID         *string
	}

	var allVisitorData []visitorData

	for rows.Next() {
		var vd visitorData
		if err := rows.Scan(
			&vd.VisitorID,
			&vd.EventCount,
			&vd.FirstSeen,
			&vd.LastSeen,
			&vd.LocationCountryISO,
			&vd.LocationCity,
			&vd.DeviceType,
			&vd.BrowserName,
			&vd.UserID,
			&vd.ExternalID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan visitor data: %w", err)
		}
		allVisitorData = append(allVisitorData, vd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating visitor rows: %w", err)
	}

	type groupKey struct {
		UserID     string
		ExternalID string
	}

	groupMap := make(map[groupKey]*types.TopVisitor)
	ungroupedVisitors := make([]*types.TopVisitor, 0)

	for _, vd := range allVisitorData {
		var key groupKey
		var canGroup bool

		if vd.UserID != nil && *vd.UserID != "" {
			key.UserID = *vd.UserID
			canGroup = true
		} else if vd.ExternalID != nil && *vd.ExternalID != "" {
			key.ExternalID = *vd.ExternalID
			canGroup = true
		}

		if canGroup {
			if existing, exists := groupMap[key]; exists {
				existing.VisitorIDs = append(existing.VisitorIDs, vd.VisitorID)
				existing.IsGrouped = true
				existing.EventCount += vd.EventCount

				if vd.FirstSeen.Before(existing.FirstSeen) {
					existing.FirstSeen = vd.FirstSeen
				}
				if vd.LastSeen.After(existing.LastSeen) {
					existing.LastSeen = vd.LastSeen
				}
			} else {
				enhanced := &types.TopVisitor{
					UserID:             vd.UserID,
					ExternalID:         vd.ExternalID,
					VisitorIDs:         []string{vd.VisitorID},
					IsGrouped:          false,
					EventCount:         vd.EventCount,
					FirstSeen:          vd.FirstSeen,
					LastSeen:           vd.LastSeen,
					LocationCountryISO: vd.LocationCountryISO,
					LocationCity:       vd.LocationCity,
					DeviceType:         vd.DeviceType,
					BrowserName:        vd.BrowserName,
				}

				groupMap[key] = enhanced
			}
		} else {
			enhanced := &types.TopVisitor{
				VisitorIDs:         []string{vd.VisitorID},
				IsGrouped:          false,
				EventCount:         vd.EventCount,
				FirstSeen:          vd.FirstSeen,
				LastSeen:           vd.LastSeen,
				LocationCountryISO: vd.LocationCountryISO,
				LocationCity:       vd.LocationCity,
				DeviceType:         vd.DeviceType,
				BrowserName:        vd.BrowserName,
			}

			ungroupedVisitors = append(ungroupedVisitors, enhanced)
		}
	}

	result := make([]types.TopVisitor, 0)

	for _, enhanced := range groupMap {
		result = append(result, *enhanced)
	}

	for _, enhanced := range ungroupedVisitors {
		result = append(result, *enhanced)
	}

	type sortable struct {
		visitor types.TopVisitor
	}
	sortableList := make([]sortable, len(result))
	for i, v := range result {
		sortableList[i] = sortable{visitor: v}
	}

	for i := 0; i < len(sortableList); i++ {
		for j := i + 1; j < len(sortableList); j++ {
			if sortableList[j].visitor.EventCount > sortableList[i].visitor.EventCount {
				sortableList[i], sortableList[j] = sortableList[j], sortableList[i]
			}
		}
	}

	finalResult := make([]types.TopVisitor, 0)
	limit := filter.Limit
	if limit == 0 {
		limit = 50 // default limit
	}
	for i := 0; i < len(sortableList) && i < limit; i++ {
		finalResult = append(finalResult, sortableList[i].visitor)
	}

	if finalResult == nil {
		finalResult = []types.TopVisitor{}
	}

	return finalResult, nil
}

func (a *AnalyticsData) GetVisitorProfile(ctx *ctx.Ctx, filter *filters.VisitorProfileFilter) (*types.VisitorProfileResponse, error) {
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
	row := a.clickDb.Db().QueryRow(ctx, summaryQuery, filter.ProjectID, filter.VisitorID)
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

	firstEventRow := a.clickDb.Db().QueryRow(ctx, firstEventQuery, filter.ProjectID, filter.VisitorID)
	if err := firstEventRow.Scan(&profile.FirstTrafficOrigin, &profile.FirstReferrerURL); err != nil {
		profile.FirstTrafficOrigin = nil
		profile.FirstReferrerURL = nil
	}

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

	timeRows, err := a.clickDb.Db().Query(ctx, eventsOverTimeQuery, filter.ProjectID, filter.VisitorID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query events over time: %w", err)
	}
	defer clickhouse.EnsureClosed(timeRows)

	var eventsOverTime []types.EventsOverTimeDataPoint
	for timeRows.Next() {
		var dp types.EventsOverTimeDataPoint
		if err := timeRows.Scan(&dp.Timestamp, &dp.EventCount); err != nil {
			return nil, fmt.Errorf("failed to scan events over time row: %w", err)
		}
		eventsOverTime = append(eventsOverTime, dp)
	}
	profile.EventsOverTime = eventsOverTime

	// Get identity info from events table
	identityQuery := `
		SELECT
			anyIf(user_id, user_id IS NOT NULL AND user_id != '') as user_id,
			anyIf(external_id, external_id IS NOT NULL AND external_id != '') as external_id
		FROM events
		WHERE project_id = ?
			AND visitor_id = ?
		LIMIT 1
	`

	var userID, externalID *string
	identityRow := a.clickDb.Db().QueryRow(ctx, identityQuery, filter.ProjectID, *filter.VisitorID)
	if err := identityRow.Scan(&userID, &externalID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		fmt.Printf("Warning: failed to fetch visitor identity from events: %v\n", err)
	}

	profile.UserID = userID
	profile.ExternalID = externalID
	profile.IsIdentified = (userID != nil && *userID != "") || (externalID != nil && *externalID != "")

	return &profile, nil
}

func (a *AnalyticsData) GetChurnRate(ctx *ctx.Ctx, filter *filters.SectionFilter, churnThresholdDays int) (*types.ChurnRateResponse, error) {
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

	row := a.clickDb.Db().QueryRow(ctx, query, filter.ProjectID, filter.TimeRange.Start)
	if err := row.Scan(&response.TotalUsers, &response.ChurnedUsers, &response.ChurnRatePercent); err != nil {
		return nil, fmt.Errorf("failed to query churn rate: %w", err)
	}

	return &response, nil
}

func (a *AnalyticsData) GetCohortAnalysis(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.CohortAnalysisResponse, error) {
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

	rows, err := a.clickDb.Db().Query(ctx, query, filter.ProjectID, filter.TimeRange.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to query cohort analysis: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

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

	if cohorts == nil {
		cohorts = []types.CohortData{}
	}

	return &types.CohortAnalysisResponse{
		Cohorts: cohorts,
	}, nil
}

