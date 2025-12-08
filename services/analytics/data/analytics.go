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
	ingestionData "zori/services/ingestion/data"
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

func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
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

	var allRelatedVisitorIDs []string
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

		allRelatedVisitorIDs, err = a.visitorRepository.GetAllVisitorIDsByIdentities(ctx, filter.ProjectID, visitorIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get all related visitor IDs: %w", err)
		}

		allIdentities, err := a.visitorRepository.GetVisitorsByIDs(ctx, allRelatedVisitorIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get all related visitor identities: %w", err)
		}
		for _, identity := range allIdentities {
			if _, exists := visitorIdentityMap[identity.VisitorID]; !exists {
				visitorIdentityMap[identity.VisitorID] = &visitorIdentity{
					UserID:     identity.UserID,
					ExternalID: identity.ExternalID,
					Email:      identity.Email,
					Name:       identity.Name,
				}
			}
		}
	} else {
		allRelatedVisitorIDs = []string{}
	}

	earliestSeenQuery := `
		SELECT
			visitor_id,
			min(client_timestamp_utc) as earliest_seen
		FROM events
		WHERE project_id = ?
			AND visitor_id IN (` + clickhouse.BuildPlaceholders(len(allRelatedVisitorIDs)) + `)
		GROUP BY visitor_id
	`

	earliestSeenArgs := []interface{}{filter.ProjectID}
	for _, vid := range allRelatedVisitorIDs {
		earliestSeenArgs = append(earliestSeenArgs, vid)
	}

	type earliestSeenData struct {
		VisitorID    string
		EarliestSeen time.Time
	}

	earliestSeenMap := make(map[string]time.Time)
	if len(allRelatedVisitorIDs) > 0 {
		earliestRows, err := a.clickDb.Db().Query(ctx, earliestSeenQuery, earliestSeenArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to query earliest seen data: %w", err)
		}
		defer clickhouse.EnsureClosed(earliestRows)

		for earliestRows.Next() {
			var esd earliestSeenData
			if err := earliestRows.Scan(&esd.VisitorID, &esd.EarliestSeen); err != nil {
				return nil, fmt.Errorf("failed to scan earliest seen data: %w", err)
			}
			earliestSeenMap[esd.VisitorID] = esd.EarliestSeen
		}

		if err := earliestRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating earliest seen rows: %w", err)
		}
	}

	allVisitorIDsForPayments := make(map[string]bool)
	for _, vid := range visitorIDs {
		allVisitorIDsForPayments[vid] = true
	}
	for _, vid := range allRelatedVisitorIDs {
		allVisitorIDsForPayments[vid] = true
	}

	visitorIDListForPayments := make([]string, 0, len(allVisitorIDsForPayments))
	for vid := range allVisitorIDsForPayments {
		visitorIDListForPayments = append(visitorIDListForPayments, vid)
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
			AND visitor_id IN (` + clickhouse.BuildPlaceholders(len(visitorIDListForPayments)) + `)
		GROUP BY visitor_id
	`

	paymentArgs := []interface{}{filter.ProjectID}
	for _, vid := range visitorIDListForPayments {
		paymentArgs = append(paymentArgs, vid)
	}

	type paymentData struct {
		VisitorID        string
		DistinctPayments uint64
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
		defer clickhouse.EnsureClosed(paymentRows)

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

				for _, relatedID := range allRelatedVisitorIDs {
					if relatedID == vd.VisitorID {
						continue // Already checked above
					}

					if relatedIdentity, exists := visitorIdentityMap[relatedID]; exists {
						matchesIdentity := false
						if identity.UserID != nil && *identity.UserID != "" && relatedIdentity.UserID != nil && *relatedIdentity.UserID == *identity.UserID {
							matchesIdentity = true
						} else if identity.ExternalID != nil && *identity.ExternalID != "" && relatedIdentity.ExternalID != nil && *relatedIdentity.ExternalID == *identity.ExternalID {
							matchesIdentity = true
						} else if identity.Email != nil && *identity.Email != "" && relatedIdentity.Email != nil && *relatedIdentity.Email == *identity.Email {
							matchesIdentity = true
						}

						if matchesIdentity {
							if !contains(enhanced.VisitorIDs, relatedID) {
								enhanced.VisitorIDs = append(enhanced.VisitorIDs, relatedID)
							}

							if pd, hasPayment := paymentMap[relatedID]; hasPayment {
								enhanced.DistinctPayments += pd.DistinctPayments
								enhanced.TotalRevenue += float64(pd.TotalAmount) / 100.0

								if enhanced.FirstPaymentDate == nil || pd.FirstPaymentDate.Before(*enhanced.FirstPaymentDate) {
									enhanced.FirstPaymentDate = &pd.FirstPaymentDate
								}
								if enhanced.Currency == nil {
									enhanced.Currency = &pd.Currency
								}
							}
						}
					}
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
		var absoluteEarliestSeen time.Time
		for _, vid := range enhanced.VisitorIDs {
			if earliestSeen, exists := earliestSeenMap[vid]; exists {
				if absoluteEarliestSeen.IsZero() || earliestSeen.Before(absoluteEarliestSeen) {
					absoluteEarliestSeen = earliestSeen
				}
			}
		}

		if !absoluteEarliestSeen.IsZero() && absoluteEarliestSeen.Before(enhanced.FirstSeen) {
			enhanced.FirstSeen = absoluteEarliestSeen
		}

		if enhanced.FirstPaymentDate != nil {
			timeToFirstPurchase := enhanced.FirstPaymentDate.Sub(enhanced.FirstSeen).Seconds()
			if timeToFirstPurchase >= 0 {
				enhanced.TimeToFirstPurchaseSeconds = &timeToFirstPurchase
			}
		}
		result = append(result, *enhanced)
	}

	for _, enhanced := range ungroupedVisitors {
		for _, vid := range enhanced.VisitorIDs {
			if earliestSeen, exists := earliestSeenMap[vid]; exists {
				if earliestSeen.Before(enhanced.FirstSeen) {
					enhanced.FirstSeen = earliestSeen
				}
			}
		}

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

func (a *AnalyticsData) GetLLMTraces(ctx *ctx.Ctx, req *types.LLMTracesListRequest) ([]types.LLMTraceItem, uint64, error) {
	// Build WHERE conditions for generations table
	genWhereConditions := []string{"g.project_id = ?", "g.start_time >= ?"}
	args := []interface{}{req.ProjectID, req.TimeRange.Start}

	if req.Model != nil && *req.Model != "" {
		genWhereConditions = append(genWhereConditions, "g.model = ?")
		args = append(args, *req.Model)
	}

	genWhereClause := "WHERE " + strings.Join(genWhereConditions, " AND ")

	// Build WHERE conditions for traces table (for LEFT JOIN filtering)
	traceConditions := []string{}
	if req.Name != nil && *req.Name != "" {
		traceConditions = append(traceConditions, "t.name = ?")
		args = append(args, *req.Name)
	}
	if req.UserID != nil && *req.UserID != "" {
		traceConditions = append(traceConditions, "t.user_id = ?")
		args = append(args, *req.UserID)
	}
	if req.SessionID != nil && *req.SessionID != "" {
		traceConditions = append(traceConditions, "t.session_id = ?")
		args = append(args, *req.SessionID)
	}

	traceFilterClause := ""
	if len(traceConditions) > 0 {
		traceFilterClause = " AND " + strings.Join(traceConditions, " AND ")
	}

	countQuery := fmt.Sprintf(`
		SELECT count(DISTINCT g.trace_id)
		FROM llm_generations g
		LEFT JOIN llm_traces t ON g.project_id = t.project_id AND g.trace_id = t.trace_id
		%s%s
	`, genWhereClause, traceFilterClause)

	var totalCount uint64
	countRow := a.clickDb.Db().QueryRow(ctx, countQuery, args...)
	if err := countRow.Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT
			g.trace_id,
			any(t.name) as name,
			any(t.user_id) as user_id,
			any(t.session_id) as session_id,
			any(t.release) as release,
			any(t.version) as version,
			any(t.input) as input,
			any(t.output) as output,
			any(t.metadata) as metadata,
			any(t.tags) as tags,
			any(t.public) as public,
			min(g.start_time) as timestamp,
			min(g.created_at) as created_at,
			max(g.updated_at) as updated_at,
			count(g.generation_id) as generation_count,
			sum(g.total_cost) as total_cost,
			sum(g.total_tokens) as total_tokens,
			sum(g.input_tokens) as input_tokens,
			sum(g.output_tokens) as output_tokens,
			avg(g.latency_ms) as avg_latency_ms,
			groupUniqArray(g.model) as models
		FROM llm_generations g
		LEFT JOIN llm_traces t ON g.project_id = t.project_id AND g.trace_id = t.trace_id
		%s%s
		GROUP BY g.trace_id
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, genWhereClause, traceFilterClause)

	queryArgs := make([]interface{}, len(args), len(args)+2)
	copy(queryArgs, args)
	queryArgs = append(queryArgs, req.Limit, req.Offset)

	rows, err := a.clickDb.Db().Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query LLM traces: %w", err)
	}
	defer clickhouse.EnsureClosed(rows)

	var traces []types.LLMTraceItem
	for rows.Next() {
		var trace types.LLMTraceItem
		var models []string
		if err := rows.Scan(
			&trace.TraceID,
			&trace.Name,
			&trace.UserID,
			&trace.SessionID,
			&trace.Release,
			&trace.Version,
			&trace.Input,
			&trace.Output,
			&trace.Metadata,
			&trace.Tags,
			&trace.Public,
			&trace.Timestamp,
			&trace.CreatedAt,
			&trace.UpdatedAt,
			&trace.GenerationCount,
			&trace.TotalCost,
			&trace.TotalTokens,
			&trace.InputTokens,
			&trace.OutputTokens,
			&trace.AvgLatencyMs,
			&models,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}
		// Filter out empty strings from models array
		filteredModels := make([]string, 0, len(models))
		for _, m := range models {
			if m != "" {
				filteredModels = append(filteredModels, m)
			}
		}
		trace.Models = filteredModels
		traces = append(traces, trace)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	return traces, totalCount, nil
}

func (a *AnalyticsData) GetLLMTraceFilterOptions(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.LLMTraceFilterOptionsResponse, error) {
	whereClause := "WHERE project_id = ?"
	args := []interface{}{filter.ProjectID}

	if filter.TimeRange != nil {
		whereClause += " AND timestamp >= ?"
		args = append(args, filter.TimeRange.Start)
	}

	namesQuery := fmt.Sprintf(`
		SELECT DISTINCT name
		FROM llm_traces
		%s
			AND name IS NOT NULL
			AND name != ''
		ORDER BY name
		LIMIT 1000
	`, whereClause)

	namesRows, err := a.clickDb.Db().Query(ctx, namesQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query trace names: %w", err)
	}
	defer clickhouse.EnsureClosed(namesRows)

	var names []string
	for namesRows.Next() {
		var name string
		if err := namesRows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan name: %w", err)
		}
		names = append(names, name)
	}

	if err := namesRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating names: %w", err)
	}

	userIDsQuery := fmt.Sprintf(`
		SELECT DISTINCT user_id
		FROM llm_traces
		%s
			AND user_id IS NOT NULL
			AND user_id != ''
		ORDER BY user_id
		LIMIT 1000
	`, whereClause)

	userIDsRows, err := a.clickDb.Db().Query(ctx, userIDsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user_ids: %w", err)
	}
	defer clickhouse.EnsureClosed(userIDsRows)

	var userIDs []string
	for userIDsRows.Next() {
		var userID string
		if err := userIDsRows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user_id: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := userIDsRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user_ids: %w", err)
	}

	sessionIDsQuery := fmt.Sprintf(`
		SELECT DISTINCT session_id
		FROM llm_traces
		%s
			AND session_id IS NOT NULL
			AND session_id != ''
		ORDER BY session_id
		LIMIT 1000
	`, whereClause)

	sessionIDsRows, err := a.clickDb.Db().Query(ctx, sessionIDsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query session_ids: %w", err)
	}
	defer clickhouse.EnsureClosed(sessionIDsRows)

	var sessionIDs []string
	for sessionIDsRows.Next() {
		var sessionID string
		if err := sessionIDsRows.Scan(&sessionID); err != nil {
			return nil, fmt.Errorf("failed to scan session_id: %w", err)
		}
		sessionIDs = append(sessionIDs, sessionID)
	}

	if err := sessionIDsRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session_ids: %w", err)
	}

	genWhereClause := "WHERE project_id = ?"
	genArgs := []interface{}{filter.ProjectID}

	if filter.TimeRange != nil {
		genWhereClause += " AND start_time >= ?"
		genArgs = append(genArgs, filter.TimeRange.Start)
	}

	modelsQuery := fmt.Sprintf(`
		SELECT DISTINCT model
		FROM llm_generations
		%s
			AND model IS NOT NULL
			AND model != ''
		ORDER BY model
		LIMIT 1000
	`, genWhereClause)

	modelsRows, err := a.clickDb.Db().Query(ctx, modelsQuery, genArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query models: %w", err)
	}
	defer clickhouse.EnsureClosed(modelsRows)

	var models []string
	for modelsRows.Next() {
		var model string
		if err := modelsRows.Scan(&model); err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}
		models = append(models, model)
	}

	if err := modelsRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating models: %w", err)
	}

	if names == nil {
		names = []string{}
	}
	if userIDs == nil {
		userIDs = []string{}
	}
	if sessionIDs == nil {
		sessionIDs = []string{}
	}
	if models == nil {
		models = []string{}
	}

	return &types.LLMTraceFilterOptionsResponse{
		Names:      names,
		UserIDs:    userIDs,
		SessionIDs: sessionIDs,
		Models:     models,
	}, nil
}
