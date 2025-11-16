package tiles

import (
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type DAUResponse struct {
	Count         uint64 `json:"count"`
	PreviousCount uint64 `json:"previous_count"`
}

type DAUTile struct {
	db *clickhouse.ClickhouseDB
}

func NewDAUTile(db *clickhouse.ClickhouseDB) *DAUTile {
	return &DAUTile{db: db}
}

func (t *DAUTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*DAUResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *DAUTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*DAUResponse, error) {
	query := t.buildQuery()

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID)

	var data DAUResponse
	err := row.Scan(&data.Count, &data.PreviousCount)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *DAUTile) buildQuery() string {
	return `
		SELECT
		    uniqIf(visitor_id, created_at >= now() - INTERVAL 1 DAY) AS count,
		    uniqIf(visitor_id, created_at >= now() - INTERVAL 2 DAY AND created_at < now() - INTERVAL 1 DAY) AS previous_count
		FROM events
		WHERE organization_id = ?
		AND project_id = ?
	`
}
