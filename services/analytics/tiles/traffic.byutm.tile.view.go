package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type UTMTrafficSourceResponse struct {
	Data []UTMTrafficSourceData `json:"data"`
}

type UTMTrafficSourceData struct {
	UTMSource       string  `json:"utm_source" ch:"utm_source"`
	UTMMedium       string  `json:"utm_medium" ch:"utm_medium"`
	UTMCampaign     string  `json:"utm_campaign" ch:"utm_campaign"`
	Count           *uint64 `json:"count" ch:"current_visits"`
	PreviousCount   *uint64 `json:"previous_count" ch:"previous_visits"`
	Revenue         *int64  `json:"revenue,omitempty" ch:"current_revenue"`
	PreviousRevenue *int64  `json:"previous_revenue,omitempty" ch:"previous_revenue"`
}

type TrafficUTMSourceTile struct {
	db *clickhouse.ClickhouseDB
}

func NewTrafficUTMSourceTile(db *clickhouse.ClickhouseDB) *TrafficUTMSourceTile {
	return &TrafficUTMSourceTile{db: db}
}

func (t *TrafficUTMSourceTile) FetchByUTM(ctx *ctx.Ctx, filter *filters.SectionFilter) (*UTMTrafficSourceResponse, error) {
	trafficByUTM, err := t.fetchUTMTrafficData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &UTMTrafficSourceResponse{
		Data: trafficByUTM,
	}, nil
}

func (t *TrafficUTMSourceTile) fetchUTMTrafficData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]UTMTrafficSourceData, error) {
	query := t.buildTrafficSourceUTMQuery(ctx, filter)

	// Query requires 4 parameters: orgID and projectID for traffic_data CTE, then again for revenue_data CTE
	queryResult, err := t.db.Db().Query(ctx, query,
		ctx.OrgID(), filter.ProjectID,  // For traffic_data CTE
		ctx.OrgID(), filter.ProjectID,  // For revenue_data CTE
	)
	if err != nil {
		return nil, err
	}
	defer clickhouse.EnsureClosed(queryResult)

	var utmTrafficSourceData []UTMTrafficSourceData

	for queryResult.Next() {
		var data UTMTrafficSourceData
		err := queryResult.ScanStruct(&data)
		if err != nil {
			return nil, err
		}
		utmTrafficSourceData = append(utmTrafficSourceData, data)
	}

	return utmTrafficSourceData, nil
}

func (t *TrafficUTMSourceTile) buildTrafficSourceUTMQuery(ctx *ctx.Ctx, filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		WITH
		-- Get traffic counts by UTM parameters from events
		traffic_data AS (
			SELECT
				ifNull(nullIf(utm_parameters['utm_source'], ''), 'NONE') AS utm_source,
				ifNull(nullIf(utm_parameters['utm_medium'], ''), 'NONE') AS utm_medium,
				ifNull(nullIf(utm_parameters['utm_campaign'], ''), 'NONE') AS utm_campaign,
				uniqIf(visitor_id, created_at > now() - %[1]s) AS current_visits,
				uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_visits
			FROM events
			WHERE events.organization_id = ?
			AND events.project_id = ?
			AND (
				utm_parameters['utm_source'] != ''
				OR utm_parameters['utm_medium'] != ''
				OR utm_parameters['utm_campaign'] != ''
			)
			GROUP BY utm_source, utm_medium, utm_campaign
		),
		-- Get revenue by UTM parameters using first-touch attribution
		revenue_data AS (
			SELECT
				utm_source,
				utm_medium,
				utm_campaign,
				sum(current_revenue) AS current_revenue,
				sum(previous_revenue) AS previous_revenue
			FROM (
				SELECT
					ifNull(nullIf(vfta.utm_source, ''), 'NONE') AS utm_source,
					ifNull(nullIf(vfta.utm_medium, ''), 'NONE') AS utm_medium,
					ifNull(nullIf(vfta.utm_campaign, ''), 'NONE') AS utm_campaign,
					sumIf(pe.amount, pe.payment_timestamp_utc > now() - %[1]s AND pe.payment_status = 'succeeded') AS current_revenue,
					sumIf(pe.amount, pe.payment_timestamp_utc BETWEEN now() - %[2]s AND now() - %[1]s AND pe.payment_status = 'succeeded') AS previous_revenue
				FROM payment_events pe
				LEFT JOIN (
					SELECT
						organization_id,
						project_id,
						visitor_id,
						argMinMerge(first_utm_source) AS utm_source,
						argMinMerge(first_utm_medium) AS utm_medium,
						argMinMerge(first_utm_campaign) AS utm_campaign
					FROM visitor_first_touch_attribution
					GROUP BY organization_id, project_id, visitor_id
				) vfta
					ON pe.visitor_id = vfta.visitor_id
					AND pe.organization_id = vfta.organization_id
					AND pe.project_id = vfta.project_id
				WHERE pe.organization_id = ?
				AND pe.project_id = ?
				AND pe.visitor_id IS NOT NULL
				GROUP BY pe.visitor_id, vfta.utm_source, vfta.utm_medium, vfta.utm_campaign
			)
			GROUP BY utm_source, utm_medium, utm_campaign
		)
		-- Combine traffic and revenue data
		SELECT
			COALESCE(t.utm_source, r.utm_source) AS utm_source,
			COALESCE(t.utm_medium, r.utm_medium) AS utm_medium,
			COALESCE(t.utm_campaign, r.utm_campaign) AS utm_campaign,
			COALESCE(t.current_visits, 0) AS current_visits,
			COALESCE(t.previous_visits, 0) AS previous_visits,
			COALESCE(r.current_revenue, 0) AS current_revenue,
			COALESCE(r.previous_revenue, 0) AS previous_revenue
		FROM traffic_data t
		FULL OUTER JOIN revenue_data r
			ON t.utm_source = r.utm_source
			AND t.utm_medium = r.utm_medium
			AND t.utm_campaign = r.utm_campaign
		WHERE COALESCE(t.current_visits, 0) > 0 OR COALESCE(r.current_revenue, 0) > 0
		ORDER BY current_visits DESC, current_revenue DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
