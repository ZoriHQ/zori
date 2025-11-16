package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type ReturnRateResponse struct {
	Rate         float64 `json:"rate"`
	PreviousRate float64 `json:"previous_rate"`
}

type ReturnRateTile struct {
	db *clickhouse.ClickhouseDB
}

func NewReturnRateTile(db *clickhouse.ClickhouseDB) *ReturnRateTile {
	return &ReturnRateTile{db: db}
}

func (t *ReturnRateTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*ReturnRateResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *ReturnRateTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*ReturnRateResponse, error) {
	query := t.buildQuery(filter)

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)

	var data ReturnRateResponse
	err := row.Scan(&data.Rate, &data.PreviousRate)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *ReturnRateTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		WITH current_visitors AS (
			SELECT visitor_id, uniq(session_id) as session_count
			FROM events
			WHERE organization_id = ?
			AND project_id = ?
			AND created_at > now() - %[1]s
			AND session_id != ''
			GROUP BY visitor_id
		),
		previous_visitors AS (
			SELECT visitor_id, uniq(session_id) as session_count
			FROM events
			WHERE organization_id = ?
			AND project_id = ?
			AND created_at BETWEEN now() - %[2]s AND now() - %[1]s
			AND session_id != ''
			GROUP BY visitor_id
		)
		SELECT
			CASE WHEN count(*) > 0
				THEN (countIf(session_count > 1) * 100.0 / count(*))
				ELSE 0
			END as rate,
			CASE WHEN (SELECT count(*) FROM previous_visitors) > 0
				THEN ((SELECT countIf(session_count > 1) FROM previous_visitors) * 100.0 / (SELECT count(*) FROM previous_visitors))
				ELSE 0
			END as previous_rate
		FROM current_visitors
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
