package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type BounceRateResponse struct {
	Rate         float64 `json:"rate" ch:"rate"`
	PreviousRate float64 `json:"previous_rate" ch:"previous_rate"`
}

type BounceRateTile struct {
	db *clickhouse.ClickhouseDB
}

func NewBounceRateTile(db *clickhouse.ClickhouseDB) *BounceRateTile {
	return &BounceRateTile{db: db}
}

func (t *BounceRateTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*BounceRateResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *BounceRateTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*BounceRateResponse, error) {
	query := t.buildQuery(filter)

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)

	var data BounceRateResponse
	err := row.ScanStruct(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *BounceRateTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		WITH current_sessions AS (
			SELECT session_id, sum(page_count) as total_pages
			FROM sessions
			WHERE organization_id = ?
			AND project_id = ?
			AND started_at > now() - %[1]s
			GROUP BY session_id
		),
		previous_sessions AS (
			SELECT session_id, sum(page_count) as total_pages
			FROM sessions
			WHERE organization_id = ?
			AND project_id = ?
			AND started_at BETWEEN now() - %[2]s AND now() - %[1]s
			GROUP BY session_id
		)
		SELECT
			CASE WHEN count(*) > 0
				THEN (countIf(total_pages <= 1) * 100.0 / count(*))
				ELSE 0
			END as rate,
			CASE WHEN (SELECT count(*) FROM previous_sessions) > 0
				THEN ((SELECT countIf(total_pages <= 1) FROM previous_sessions) * 100.0 / (SELECT count(*) FROM previous_sessions))
				ELSE 0
			END as previous_rate
		FROM current_sessions
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
