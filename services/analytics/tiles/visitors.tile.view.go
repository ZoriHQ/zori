package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type UniqueVisitorsResponse struct {
	Count         uint64 `json:"count" ch:"count"`
	PreviousCount uint64 `json:"previous_count" ch:"previous_count"`
}

type UniqueVisitorsTile struct {
	db *clickhouse.ClickhouseDB
}

func NewUniqueVisitorsTile(db *clickhouse.ClickhouseDB) *UniqueVisitorsTile {
	return &UniqueVisitorsTile{db: db}
}

func (t *UniqueVisitorsTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*UniqueVisitorsResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *UniqueVisitorsTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*UniqueVisitorsResponse, error) {
	query := t.buildQuery(filter)

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID)

	var data UniqueVisitorsResponse
	err := row.ScanStruct(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *UniqueVisitorsTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    uniqIf(visitor_id, created_at > now() - %[1]s) AS count,
		    uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM events
		WHERE organization_id = ?
		AND project_id = ?
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
