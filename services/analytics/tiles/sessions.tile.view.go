package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type UniqueSessionsResponse struct {
	Count         uint64 `json:"count"`
	PreviousCount uint64 `json:"previous_count"`
}

type UniqueSessionsTile struct {
	db *clickhouse.ClickhouseDB
}

func NewUniqueSessionsTile(db *clickhouse.ClickhouseDB) *UniqueSessionsTile {
	return &UniqueSessionsTile{db: db}
}

func (t *UniqueSessionsTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*UniqueSessionsResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *UniqueSessionsTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*UniqueSessionsResponse, error) {
	query := t.buildQuery(filter)

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID)

	var data UniqueSessionsResponse
	err := row.Scan(&data.Count, &data.PreviousCount)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *UniqueSessionsTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    uniqIf(session_id, created_at > now() - %[1]s) AS count,
		    uniqIf(session_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM events
		WHERE organization_id = ?
		AND project_id = ?
		AND session_id != ''
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
