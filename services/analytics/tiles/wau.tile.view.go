package tiles

import (
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type WAUResponse struct {
	Count         uint64 `json:"count" ch:"count"`
	PreviousCount uint64 `json:"previous_count" ch:"previous_count"`
}

type WAUTile struct {
	db *clickhouse.ClickhouseDB
}

func NewWAUTile(db *clickhouse.ClickhouseDB) *WAUTile {
	return &WAUTile{db: db}
}

func (t *WAUTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*WAUResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *WAUTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*WAUResponse, error) {
	query := t.buildQuery()

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID)

	var data WAUResponse
	err := row.ScanStruct(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *WAUTile) buildQuery() string {
	return `
		SELECT
		    uniqIf(visitor_id, created_at >= now() - INTERVAL 7 DAY) AS count,
		    uniqIf(visitor_id, created_at >= now() - INTERVAL 14 DAY AND created_at < now() - INTERVAL 7 DAY) AS previous_count
		FROM events
		WHERE organization_id = ?
		AND project_id = ?
	`
}
