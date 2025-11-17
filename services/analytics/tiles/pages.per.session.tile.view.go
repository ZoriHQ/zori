package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type PagesPerSessionResponse struct {
	AvgPages         float64 `json:"avg_pages"  ch:"avg_pages"`
	PreviousAvgPages float64 `json:"previous_avg_pages" ch:"previous_avg_pages"`
}

type PagesPerSessionTile struct {
	db *clickhouse.ClickhouseDB
}

func NewPagesPerSessionTile(db *clickhouse.ClickhouseDB) *PagesPerSessionTile {
	return &PagesPerSessionTile{db: db}
}

func (t *PagesPerSessionTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*PagesPerSessionResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *PagesPerSessionTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*PagesPerSessionResponse, error) {
	query := t.buildQuery(filter)

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)

	var data PagesPerSessionResponse
	err := row.ScanStruct(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *PagesPerSessionTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		WITH current_sessions AS (
			SELECT
				session_id,
				min(started_at) as session_start,
				sum(page_count) as total_pages
			FROM sessions
			WHERE organization_id = ?
			AND project_id = ?
			GROUP BY session_id, visitor_id, project_id, organization_id
			HAVING session_start > now() - %[1]s
		),
		current_avg AS (
			SELECT AVG(total_pages) as avg_pages
			FROM current_sessions
		),
		previous_sessions AS (
			SELECT
				session_id,
				min(started_at) as session_start,
				sum(page_count) as total_pages
			FROM sessions
			WHERE organization_id = ?
			AND project_id = ?
			GROUP BY session_id, visitor_id, project_id, organization_id
			HAVING session_start BETWEEN now() - %[2]s AND now() - %[1]s
		),
		previous_avg AS (
			SELECT AVG(total_pages) as avg_pages
			FROM previous_sessions
		)
		SELECT
			if(isNaN((SELECT avg_pages FROM current_avg)), 0, COALESCE((SELECT avg_pages FROM current_avg), 0)) as avg_pages,
			if(isNaN((SELECT avg_pages FROM previous_avg)), 0, COALESCE((SELECT avg_pages FROM previous_avg), 0)) as previous_avg_pages
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
