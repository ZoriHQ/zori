package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type TimeBetweenVisitsResponse struct {
	AvgHours         float64 `json:"avg_hours"`
	PreviousAvgHours float64 `json:"previous_avg_hours"`
}

type TimeBetweenVisitsTile struct {
	db *clickhouse.ClickhouseDB
}

func NewTimeBetweenVisitsTile(db *clickhouse.ClickhouseDB) *TimeBetweenVisitsTile {
	return &TimeBetweenVisitsTile{db: db}
}

func (t *TimeBetweenVisitsTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*TimeBetweenVisitsResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *TimeBetweenVisitsTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*TimeBetweenVisitsResponse, error) {
	query := t.buildQuery(filter)

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)

	var data TimeBetweenVisitsResponse
	err := row.Scan(&data.AvgHours, &data.PreviousAvgHours)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *TimeBetweenVisitsTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		WITH current_session_gaps AS (
			SELECT
				s1.visitor_id,
				dateDiff('hour', s1.started_at, s2.started_at) as hours_between
			FROM sessions s1
			JOIN sessions s2
				ON s1.visitor_id = s2.visitor_id
				AND s2.started_at > s1.started_at
			WHERE s1.organization_id = ?
			AND s1.project_id = ?
			AND s1.started_at > now() - %[1]s
		),
		previous_session_gaps AS (
			SELECT
				s1.visitor_id,
				dateDiff('hour', s1.started_at, s2.started_at) as hours_between
			FROM sessions s1
			JOIN sessions s2
				ON s1.visitor_id = s2.visitor_id
				AND s2.started_at > s1.started_at
			WHERE s1.organization_id = ?
			AND s1.project_id = ?
			AND s1.started_at BETWEEN now() - %[2]s AND now() - %[1]s
		)
		SELECT
			COALESCE(AVG(hours_between), 0) as avg_hours,
			COALESCE((SELECT AVG(hours_between) FROM previous_session_gaps), 0) as previous_avg_hours
		FROM current_session_gaps
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
