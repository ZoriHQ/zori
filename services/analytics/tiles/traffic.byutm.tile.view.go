package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type UTMCampaignSourceData struct {
	UTMSource       string `json:"utm_source" ch:"utm_source"`
	UTMMedium       string `json:"utm_medium" ch:"utm_medium"`
	UTMCampaign     string `json:"utm_campaign" ch:"utm_campaign"`
	Count           uint   `json:"count" ch:"count"`
	PreviousCount   uint   `json:"previous_count" ch:"previous_count"`
	Revenue         uint   `json:"revenue" ch:"current_revenue"`
	PreviousRevenue uint   `json:"previous_revenue" ch:"previous_revenue"`
}

type TrafficSourceTile struct {
	db *clickhouse.ClickhouseDB
}

func NewTrafficSourceTile(db *clickhouse.ClickhouseDB) *TrafficSourceTile {
	return &TrafficSourceTile{db: db}
}

func (t *TrafficSourceTile) FetchByReferer(ctx *ctx.Ctx, filter *filters.SectionFilter) (*RefererTrafficSourceResponse, error) {
	trafficByReferer, err := t.fetchRefererTrafficData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &RefererTrafficSourceResponse{
		Data: trafficByReferer,
	}, nil
}

func (t *TrafficSourceTile) FetchByCountry(ctx *ctx.Ctx, filter *filters.SectionFilter) (*CountryTrafficSourceResponse, error) {
	trafficByCountry, err := t.fetchCountryTrafficData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &CountryTrafficSourceResponse{
		Data: trafficByCountry,
	}, nil
}

func (t *TrafficSourceTile) fetchRefererTrafficData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]RefererTrafficSourceData, error) {
	query := t.buildTrafficSourceRefererQuery(ctx, filter)

	fmt.Println("Traffic souirce referere query: ", query)

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

func (t *TrafficSourceTile) fetchCountryTrafficData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]*CountryTrafficSourceData, error) {
	query := t.buildTrafficSourceCountryQuery(ctx, filter)

	queryResult, err := t.db.Db().Query(ctx, query, ctx.OrgID(), filter.ProjectID)
	if err != nil {
		return nil, err
	}
	defer clickhouse.EnsureClosed(queryResult)

	var countryTrafficSourceData []*CountryTrafficSourceData
	for queryResult.Next() {
		var data CountryTrafficSourceData
		err := queryResult.ScanStruct(&data)
		if err != nil {
			return nil, err
		}
		countryTrafficSourceData = append(countryTrafficSourceData, &data)
	}

	return countryTrafficSourceData, nil
}

func (t *TrafficSourceTile) buildTrafficSourceRefererQuery(ctx *ctx.Ctx, filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    ifNull(referrer_domain, 'DIRECT/NONE') AS referer_url,
		    uniqIf(visitor_id, created_at > now() - %[1]s) AS current_visits,
		    uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_visits
		FROM events
		WHERE events.organization_id = ?
		AND events.project_id = ?
		GROUP BY referrer_domain
		ORDER BY current_visits DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}

func (t *TrafficSourceTile) buildTrafficSourceCountryQuery(ctx *ctx.Ctx, filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    ifNull(location_country_iso, 'UNKNOWN') AS country,
		    uniqIf(visitor_id, created_at > now() - %[1]s) AS count,
		    uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM events
		WHERE events.organization_id = ?
		AND events.project_id = ?
		GROUP BY location_country_iso
		HAVING count > 0
		ORDER BY count DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}

func (t *TrafficSourceTile) buildTrafficSourceRevenueQuery(filter *filters.SectionFilter) string {
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
