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
	UTMSource     string  `json:"utm_source" ch:"utm_source"`
	UTMMedium     string  `json:"utm_medium" ch:"utm_medium"`
	UTMCampaign   string  `json:"utm_campaign" ch:"utm_campaign"`
	Count         *uint64 `json:"count" ch:"current_visits"`
	PreviousCount *uint64 `json:"previous_count" ch:"previous_visits"`
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

	queryResult, err := t.db.Db().Query(ctx, query,
		ctx.OrgID(), filter.ProjectID,
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
		SELECT
			ifNull(nullIf(utm_parameters['utm_source'], ''), 'NONE') AS utm_source,
			ifNull(nullIf(utm_parameters['utm_medium'], ''), 'NONE') AS utm_medium,
			ifNull(nullIf(utm_parameters['utm_campaign'], ''), 'NONE') AS utm_campaign,
			uniqIf(visitor_id, created_at > now() - %[1]s) AS current_visits,
			uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_visits
		FROM events
		WHERE events.organization_id = ?
		AND events.project_id = ?
		AND created_at > now() - %[2]s
		AND (
			utm_parameters['utm_source'] != ''
			OR utm_parameters['utm_medium'] != ''
			OR utm_parameters['utm_campaign'] != ''
		)
		GROUP BY utm_source, utm_medium, utm_campaign
		ORDER BY current_visits DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
