package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type RefererTrafficSourceResponse struct {
	Data []RefererTrafficSourceData `json:"data"`
}

type RefererTrafficSourceData struct {
	RefererURL      string  `json:"referer_url" ch:"referer_url"`
	Count           *uint64 `json:"count" ch:"current_visits"`
	PreviousCount   *uint64 `json:"previous_count" ch:"previous_visits"`
	Revenue         *uint64 `json:"revenue" ch:"current_revenue"`
	PreviousRevenue *uint64 `json:"previous_revenue" ch:"previous_revenue"`
}

type TrafficRefererSourceTile struct {
	db *clickhouse.ClickhouseDB
}

func NewTrafficRefererSourceTile(db *clickhouse.ClickhouseDB) *TrafficRefererSourceTile {
	return &TrafficRefererSourceTile{db: db}
}

func (t *TrafficRefererSourceTile) FetchByReferer(ctx *ctx.Ctx, filter *filters.SectionFilter) (*RefererTrafficSourceResponse, error) {
	trafficByReferer, err := t.fetchRefererTrafficData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &RefererTrafficSourceResponse{
		Data: trafficByReferer,
	}, nil
}

func (t *TrafficRefererSourceTile) fetchRefererTrafficData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]RefererTrafficSourceData, error) {
	query := t.buildTrafficSourceRefererQuery(ctx, filter)

	queryResult, err := t.db.Db().Query(ctx, query, ctx.OrgID(), filter.ProjectID)
	if err != nil {
		return nil, err
	}
	defer clickhouse.EnsureClosed(queryResult)

	var refererTrafficSourceData []RefererTrafficSourceData

	for queryResult.Next() {
		var data RefererTrafficSourceData
		err := queryResult.ScanStruct(&data)
		if err != nil {
			return nil, err
		}
		refererTrafficSourceData = append(refererTrafficSourceData, data)
	}

	return refererTrafficSourceData, nil
}

func (t *TrafficRefererSourceTile) buildTrafficSourceRefererQuery(ctx *ctx.Ctx, filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    ifNull(nullIf(referrer_domain, ''), 'DIRECT/NONE') AS referer_url,
		    uniqIf(visitor_id, created_at > now() - %[1]s) AS current_visits,
		    uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_visits
		FROM events
		WHERE events.organization_id = ?
		AND events.project_id = ?
		GROUP BY referer_url
		ORDER BY current_visits DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}

func (t *TrafficRefererSourceTile) buildTrafficSourceRevenueQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		WITH visitor_attribution AS (
		    SELECT
		        visitor_id,
		        argMax(referrer_domain, client_timestamp_utc) AS last_referrer
		    FROM events
		    GROUP BY visitor_id
		)
		SELECT
		    ifNull(va.last_referrer, 'DIRECT/NONE') AS referer_url,
			countDistinct(pe.visitor_id) AS visitors,
		    sumIf(pe.amount, pe.payment_timestamp_utc > now() - %[1]s) AS current_revenue,
		    uniqIf(pe.visitor_id, pe.payment_timestamp_utc > now() - %[1]s) AS current_paying_visitors,
		    countIf(pe.payment_id, pe.payment_timestamp_utc > now() - %[1]s) AS current_payments
		FROM payment_events pe
		LEFT JOIN visitor_attribution va ON pe.visitor_id = va.visitor_id
		WHERE pe.visitor_id IS NOT NULL
		GROUP BY referer_url
		ORDER BY current_revenue DESC;
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
