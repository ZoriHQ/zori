package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type CountryTrafficSourceResponse struct {
	Data []*CountryTrafficSourceData `json:"data"`
}

type CountryTrafficSourceData struct {
	Country       string `json:"country" ch:"country"`
	Count         uint64 `json:"count" ch:"count"`
	PreviousCount uint64 `json:"previous_count" ch:"previous_count"`
}

type TrafficCountrySourceTile struct {
	db *clickhouse.ClickhouseDB
}

func NewTrafficCountrySourceTile(db *clickhouse.ClickhouseDB) *TrafficCountrySourceTile {
	return &TrafficCountrySourceTile{db: db}
}

func (t *TrafficCountrySourceTile) FetchByCountry(ctx *ctx.Ctx, filter *filters.SectionFilter) (*CountryTrafficSourceResponse, error) {
	trafficByCountry, err := t.fetchCountryTrafficData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &CountryTrafficSourceResponse{
		Data: trafficByCountry,
	}, nil
}

func (t *TrafficCountrySourceTile) fetchCountryTrafficData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]*CountryTrafficSourceData, error) {
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

func (t *TrafficCountrySourceTile) buildTrafficSourceCountryQuery(ctx *ctx.Ctx, filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    ifNull(nullIf(location_country_iso, ''), 'UNKNOWN') AS country,
		    uniqIf(visitor_id, created_at > now() - %[1]s) AS count,
		    uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM events
		WHERE events.organization_id = ?
		AND events.project_id = ?
		AND created_at > now() - %[2]s
		GROUP BY country
		HAVING count > 0
		ORDER BY count DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
