package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type SessionDurationResponse struct {
	AvgDuration         float64 `json:"avg_duration" ch:"avg_duration"`
	PreviousAvgDuration float64 `json:"previous_avg_duration" ch:"previous_avg_duration"`
}

type SessionDurationTile struct {
	db *clickhouse.ClickhouseDB
}

func NewSessionDurationTile(db *clickhouse.ClickhouseDB) *SessionDurationTile {
	return &SessionDurationTile{db: db}
}

func (t *SessionDurationTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*SessionDurationResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *SessionDurationTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*SessionDurationResponse, error) {
	query := t.buildQuery(filter)

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)

	var data SessionDurationResponse
	err := row.ScanStruct(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t *SessionDurationTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		WITH current_sessions AS (
			SELECT
				session_id,
				min(started_at) as session_start,
				max(ended_at) as session_end
			FROM sessions
			WHERE organization_id = ?
			AND project_id = ?
			GROUP BY session_id, visitor_id, project_id, organization_id
			HAVING session_start > now() - %[1]s
		),
		current_avg AS (
			SELECT AVG(dateDiff('second', session_start, session_end)) as avg_duration
			FROM current_sessions
		),
		previous_sessions AS (
			SELECT
				session_id,
				min(started_at) as session_start,
				max(ended_at) as session_end
			FROM sessions
			WHERE organization_id = ?
			AND project_id = ?
			GROUP BY session_id, visitor_id, project_id, organization_id
			HAVING session_start BETWEEN now() - %[2]s AND now() - %[1]s
		),
		previous_avg AS (
			SELECT AVG(dateDiff('second', session_start, session_end)) as avg_duration
			FROM previous_sessions
		)
		SELECT
			if(isNaN((SELECT avg_duration FROM current_avg)), 0, COALESCE((SELECT avg_duration FROM current_avg), 0)) as avg_duration,
			if(isNaN((SELECT avg_duration FROM previous_avg)), 0, COALESCE((SELECT avg_duration FROM previous_avg), 0)) as previous_avg_duration
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
