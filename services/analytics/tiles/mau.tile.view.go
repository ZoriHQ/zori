package tiles

import (
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type MAUResponse struct {
	Count         uint64 `json:"count" ch:"count"`
	PreviousCount uint64 `json:"previous_count" ch:"previous_count"`
}

type MAUTile struct {
	db *clickhouse.ClickhouseDB
}

func NewMAUTile(db *clickhouse.ClickhouseDB) *MAUTile {
	return &MAUTile{db: db}
}

func (t *MAUTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*MAUResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *MAUTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*MAUResponse, error) {
	query := t.buildQuery()

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID)

	var data MAUResponse
	err := row.ScanStruct(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *MAUTile) buildQuery() string {
	return `
		SELECT
		    uniqIf(visitor_id, created_at >= now() - INTERVAL 30 DAY) AS count,
		    uniqIf(visitor_id, created_at >= now() - INTERVAL 60 DAY AND created_at < now() - INTERVAL 30 DAY) AS previous_count
		FROM events
		WHERE organization_id = ?
		AND project_id = ?
	`
}
