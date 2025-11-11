package data

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
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
	`, filter.TimeRange.IntervalFunction)

	rows, err := a.clickDb.Db().Query(ctx, query, filter.ProjectID, filter.TimeRange.Start)
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

	if dataPoints == nil {
		dataPoints = []types.VisitorDataPoint{}
	}

	return dataPoints, nil
}

func (a *AnalyticsData) GetUniqueVisitorsByOrigin(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]types.OriginDataPoint, error) {
	query := `
		WITH visitors_in_period AS (
			SELECT DISTINCT visitor_id
			FROM events
			WHERE project_id = ?
				AND client_timestamp_utc >= ?
				AND client_timestamp_utc <= now()
		),
		total_count AS (
			SELECT COUNT(DISTINCT visitor_id) as total
			FROM visitors_in_period
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
			CASE
				WHEN (SELECT total FROM total_count) > 0
				THEN uniq(vo.visitor_id) * 100.0 / (SELECT total FROM total_count)
				ELSE 0
			END as percentage
		FROM visitor_origins vo
		GROUP BY vo.origin
		ORDER BY unique_visitors DESC
		LIMIT 20
	`

	rows, err := a.clickDb.Db().Query(ctx, query,
		filter.ProjectID, filter.TimeRange.Start,
		filter.ProjectID,
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

	if dataPoints == nil {
		dataPoints = []types.OriginDataPoint{}
	}

	return dataPoints, nil
}

func (a *AnalyticsData) GetUniqueVisitorsByCountry(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]types.CountryDataPoint, error) {
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
			CASE
				WHEN (SELECT total_visitors FROM totals) > 0
				THEN uniq(visitor_id) * 100.0 / (SELECT total_visitors FROM totals)
				ELSE 0
			END as percentage
		FROM events
		WHERE project_id = ?
			AND client_timestamp_utc >= ?
			AND client_timestamp_utc <= now()
		GROUP BY country_code
		ORDER BY unique_visitors DESC
		LIMIT 50
	`

	rows, err := a.clickDb.Db().Query(ctx, query, filter.ProjectID, filter.TimeRange.Start, filter.ProjectID, filter.TimeRange.Start)
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

	if dataPoints == nil {
		dataPoints = []types.CountryDataPoint{}
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

func (a *AnalyticsData) GetEventFilterOptions(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.EventFilterOptionsResponse, error) {
	whereClause := "WHERE project_id = ?"
	args := []interface{}{filter.ProjectID}

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

	eventNamesQuery := fmt.Sprintf(`
		SELECT DISTINCT event_name
		FROM events
		%s
			AND event_name IS NOT NULL
			AND event_name != ''
		ORDER BY event_name
		LIMIT 1000
	`, whereClause)

	eventNamesRows, err := a.clickDb.Db().Query(ctx, eventNamesQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query event names: %w", err)
	}
	defer eventNamesRows.Close()

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
	visitorDataQuery := `
		SELECT
			visitor_id,
			count() as event_count,
			min(client_timestamp_utc) as first_seen,
			max(client_timestamp_utc) as last_seen,
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
	`

	rows, err := a.clickDb.Db().Query(ctx, visitorDataQuery, filter.ProjectID, filter.TimeRange.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitor data: %w", err)
	}
	defer rows.Close()

	type visitorData struct {
		VisitorID          string
		EventCount         uint64
		FirstSeen          time.Time
		LastSeen           time.Time
		LocationCountryISO *string
		LocationCity       *string
		DeviceType         *string
		BrowserName        *string
	}

	var allVisitorData []visitorData
	visitorIDs := make([]string, 0)

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
		); err != nil {
			return nil, fmt.Errorf("failed to scan visitor data: %w", err)
		}
		allVisitorData = append(allVisitorData, vd)
		visitorIDs = append(visitorIDs, vd.VisitorID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating visitor rows: %w", err)
	}

	type visitorIdentity struct {
		UserID     *string
		ExternalID *string
		Email      *string
		Name       *string
	}
	visitorIdentityMap := make(map[string]*visitorIdentity)
	if len(visitorIDs) > 0 {
		identities, err := a.visitorRepository.GetVisitorsByIDs(ctx, visitorIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get visitor identities: %w", err)
		}
		for _, identity := range identities {
			visitorIdentityMap[identity.VisitorID] = &visitorIdentity{
				UserID:     identity.UserID,
				ExternalID: identity.ExternalID,
				Email:      identity.Email,
				Name:       identity.Name,
			}
		}
	}

	paymentQuery := `
		SELECT
			visitor_id,
			count(DISTINCT payment_id) as distinct_payments,
			sum(amount) as total_amount,
			any(currency) as currency,
			min(payment_timestamp_utc) as first_payment_date
		FROM payment_events
		WHERE project_id = ?
			AND payment_status = 'succeeded'
			AND visitor_id IS NOT NULL
			AND visitor_id IN (` + clickhouse.BuildPlaceholders(len(visitorIDs)) + `)
		GROUP BY visitor_id
	`

	paymentArgs := []interface{}{filter.ProjectID}
	for _, vid := range visitorIDs {
		paymentArgs = append(paymentArgs, vid)
	}

	type paymentData struct {
		VisitorID        string
		DistinctPayments int
		TotalAmount      int64 // in cents
		Currency         string
		FirstPaymentDate time.Time
	}

	paymentMap := make(map[string]paymentData)
	if len(visitorIDs) > 0 {
		paymentRows, err := a.clickDb.Db().Query(ctx, paymentQuery, paymentArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to query payment data: %w", err)
		}
		defer paymentRows.Close()

		for paymentRows.Next() {
			var pd paymentData
			if err := paymentRows.Scan(
				&pd.VisitorID,
				&pd.DistinctPayments,
				&pd.TotalAmount,
				&pd.Currency,
				&pd.FirstPaymentDate,
			); err != nil {
				return nil, fmt.Errorf("failed to scan payment data: %w", err)
			}
			paymentMap[pd.VisitorID] = pd
		}

		if err := paymentRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating payment rows: %w", err)
		}
	}

	type groupKey struct {
		UserID     string
		ExternalID string
		Email      string
	}

	groupMap := make(map[groupKey]*types.TopVisitor)
	ungroupedVisitors := make([]*types.TopVisitor, 0)

	for _, vd := range allVisitorData {
		identity, hasIdentity := visitorIdentityMap[vd.VisitorID]

		var key groupKey
		var canGroup bool

		if hasIdentity {
			if identity.UserID != nil && *identity.UserID != "" {
				key.UserID = *identity.UserID
				canGroup = true
			} else if identity.ExternalID != nil && *identity.ExternalID != "" {
				key.ExternalID = *identity.ExternalID
				canGroup = true
			} else if identity.Email != nil && *identity.Email != "" {
				key.Email = *identity.Email
				canGroup = true
			}
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

				if pd, hasPayment := paymentMap[vd.VisitorID]; hasPayment {
					existing.DistinctPayments += pd.DistinctPayments
					existing.TotalRevenue += float64(pd.TotalAmount) / 100.0

					if existing.FirstPaymentDate == nil || pd.FirstPaymentDate.Before(*existing.FirstPaymentDate) {
						existing.FirstPaymentDate = &pd.FirstPaymentDate
					}
				}
			} else {
				enhanced := &types.TopVisitor{
					UserID:             identity.UserID,
					ExternalID:         identity.ExternalID,
					Email:              identity.Email,
					Name:               identity.Name,
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

				if pd, hasPayment := paymentMap[vd.VisitorID]; hasPayment {
					enhanced.DistinctPayments = pd.DistinctPayments
					enhanced.TotalRevenue = float64(pd.TotalAmount) / 100.0
					enhanced.Currency = &pd.Currency
					enhanced.FirstPaymentDate = &pd.FirstPaymentDate
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

			if pd, hasPayment := paymentMap[vd.VisitorID]; hasPayment {
				enhanced.DistinctPayments = pd.DistinctPayments
				enhanced.TotalRevenue = float64(pd.TotalAmount) / 100.0
				enhanced.Currency = &pd.Currency
				enhanced.FirstPaymentDate = &pd.FirstPaymentDate
			}

			ungroupedVisitors = append(ungroupedVisitors, enhanced)
		}
	}

	result := make([]types.TopVisitor, 0)

	for _, enhanced := range groupMap {
		if enhanced.FirstPaymentDate != nil {
			timeToFirstPurchase := enhanced.FirstPaymentDate.Sub(enhanced.FirstSeen).Seconds()
			if timeToFirstPurchase >= 0 {
				enhanced.TimeToFirstPurchaseSeconds = &timeToFirstPurchase
			}
		}
		result = append(result, *enhanced)
	}

	for _, enhanced := range ungroupedVisitors {
		if enhanced.FirstPaymentDate != nil {
			timeToFirstPurchase := enhanced.FirstPaymentDate.Sub(enhanced.FirstSeen).Seconds()
			if timeToFirstPurchase >= 0 {
				enhanced.TimeToFirstPurchaseSeconds = &timeToFirstPurchase
			}
		}
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

	visitorIdentity, err := a.visitorRepository.GetVisitorByID(ctx, *filter.VisitorID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fmt.Printf("Warning: failed to fetch visitor identity: %v\n", err)
	}

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

func (a *AnalyticsData) GetUniqueVisitorsTimeline(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]types.UniqueVisitorsDataPoint, error) {
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
	`, filter.TimeRange.IntervalFunction)

	rows, err := a.clickDb.Db().Query(ctx, query, filter.ProjectID, filter.TimeRange.Start)
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

	if dataPoints == nil {
		dataPoints = []types.UniqueVisitorsDataPoint{}
	}

	return dataPoints, nil
}

func (a *AnalyticsData) GetSessionMetrics(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.SessionMetricsResponse, error) {
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
	row := a.clickDb.Db().QueryRow(ctx, query, filter.ProjectID, filter.TimeRange.Start)
	if err := row.Scan(
		&response.TotalSessions,
		&response.AverageSessionDuration,
		&response.AveragePagesPerSession,
	); err != nil {
		return nil, fmt.Errorf("failed to query session metrics: %w", err)
	}

	return &response, nil
}

func (a *AnalyticsData) GetBounceRate(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.BounceRateResponse, error) {
	overallQuery := `
		SELECT
			countIf(page_count <= 1) * 100.0 / COUNT(*) as bounce_rate
		FROM sessions
		WHERE project_id = ?
			AND started_at >= ?
			AND started_at <= now()
	`

	var overallBounceRate float64
	row := a.clickDb.Db().QueryRow(ctx, overallQuery, filter.ProjectID, filter.TimeRange.Start)
	if err := row.Scan(&overallBounceRate); err != nil {
		return nil, fmt.Errorf("failed to query overall bounce rate: %w", err)
	}

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

	rows, err := a.clickDb.Db().Query(ctx, byPageQuery, filter.ProjectID, filter.TimeRange.Start, filter.Limit)
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

	if byPageMetrics == nil {
		byPageMetrics = []types.BounceRateByPageMetric{}
	}

	return &types.BounceRateResponse{
		OverallBounceRate: overallBounceRate,
		ByPage:            byPageMetrics,
	}, nil
}

func (a *AnalyticsData) GetActiveUsers(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.ActiveUsersResponse, error) {
	query := `
		SELECT
			countIf(last_seen >= now() - INTERVAL 1 DAY) as dau,
			countIf(last_seen >= now() - INTERVAL 7 DAY) as wau,
			countIf(last_seen >= now() - INTERVAL 30 DAY) as mau
		FROM visitor_summary
		WHERE project_id = ?
	`

	var response types.ActiveUsersResponse
	row := a.clickDb.Db().QueryRow(ctx, query, filter.ProjectID)
	if err := row.Scan(&response.DAU, &response.WAU, &response.MAU); err != nil {
		return nil, fmt.Errorf("failed to query active users: %w", err)
	}

	return &response, nil
}

func (a *AnalyticsData) GetReturnRate(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.ReturnRateResponse, error) {
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
	row := a.clickDb.Db().QueryRow(ctx, query, filter.ProjectID, filter.TimeRange.Start)
	if err := row.Scan(&response.TotalUsers, &response.ReturningUsers, &response.ReturnRatePercent); err != nil {
		return nil, fmt.Errorf("failed to query return rate: %w", err)
	}

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
	avgRow := a.clickDb.Db().QueryRow(ctx, timeBetweenQuery, filter.ProjectID, filter.TimeRange.Start)
	if err := avgRow.Scan(&avgTimeBetween); err == nil {
		response.AvgTimeBetweenSessions = avgTimeBetween
	}

	return &response, nil
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

	if cohorts == nil {
		cohorts = []types.CohortData{}
	}

	return &types.CohortAnalysisResponse{
		Cohorts: cohorts,
	}, nil
}

func (a *AnalyticsData) GetDashboardMetrics(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.DashboardMetricsResponse, error) {
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

	g.Go(func() error {
		activeUsers, err := a.GetActiveUsers(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to get active users: %w", err)
		}
		res.activeUsers = activeUsers
		return nil
	})

	g.Go(func() error {
		sessionMetrics, err := a.GetSessionMetrics(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to get session metrics: %w", err)
		}
		res.sessionMetrics = sessionMetrics
		return nil
	})

	g.Go(func() error {
		bounceRate, err := a.GetBounceRate(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to get bounce rate: %w", err)
		}
		res.bounceRate = bounceRate
		return nil
	})

	g.Go(func() error {
		returnRate, err := a.GetReturnRate(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to get return rate: %w", err)
		}
		res.returnRate = returnRate
		return nil
	})

	g.Go(func() error {
		now := time.Now().UTC()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

		query := `
			SELECT COUNT(*) as sessions_today
			FROM sessions
			WHERE project_id = ?
				AND started_at >= ?
		`
		row := a.clickDb.Db().QueryRow(ctx, query, filter.ProjectID, todayStart)
		if err := row.Scan(&res.sessionsToday); err != nil {
			return fmt.Errorf("failed to get sessions today: %w", err)
		}
		return nil
	})

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
		row := a.clickDb.Db().QueryRow(ctx, query, filter.ProjectID, filter.TimeRange.Start)
		if err := row.Scan(&res.totalEvents, &res.uniqueVisitors, &res.uniqueSessions); err != nil {
			return fmt.Errorf("failed to get totals: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &types.DashboardMetricsResponse{
		DAU: res.activeUsers.DAU,
		WAU: res.activeUsers.WAU,
		MAU: res.activeUsers.MAU,

		SessionsToday:         res.sessionsToday,
		TotalSessionsInPeriod: res.sessionMetrics.TotalSessions,
		AvgSessionDuration:    res.sessionMetrics.AverageSessionDuration,
		AvgPagesPerSession:    res.sessionMetrics.AveragePagesPerSession,

		BounceRate: res.bounceRate.OverallBounceRate,
		ReturnRate: res.returnRate.ReturnRatePercent,

		TotalEvents:    res.totalEvents,
		UniqueVisitors: res.uniqueVisitors,
		UniqueSessions: res.uniqueSessions,
	}, nil
}
