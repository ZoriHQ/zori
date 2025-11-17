package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type ExitPagesData struct {
	Page          string `json:"page" ch:"page"`
	Count         uint64 `json:"count" ch:"count"`
	PreviousCount uint64 `json:"previous_count" ch:"previous_count"`
}

type ExitPagesResponse struct {
	Data []*ExitPagesData `json:"data"`
}

type ExitPagesTile struct {
	db *clickhouse.ClickhouseDB
}

func NewExitPagesTile(db *clickhouse.ClickhouseDB) *ExitPagesTile {
	return &ExitPagesTile{db: db}
}

func (t *ExitPagesTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*ExitPagesResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ExitPagesResponse{Data: data}, nil
}

func (t *ExitPagesTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]*ExitPagesData, error) {
	query := t.buildQuery(filter)

	queryResult, err := t.db.Db().Query(ctx, query, ctx.OrgID(), filter.ProjectID)
	if err != nil {
		return nil, err
	}
	defer clickhouse.EnsureClosed(queryResult)

	var exitPagesData []*ExitPagesData
	for queryResult.Next() {
		var data ExitPagesData
		err := queryResult.ScanStruct(&data)
		if err != nil {
			return nil, err
		}
		exitPagesData = append(exitPagesData, &data)
	}

	return exitPagesData, nil
}

func (t *ExitPagesTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    ifNull(nullIf(exit_page, ''), 'UNKNOWN') AS page,
		    countIf(ended_at > now() - %[1]s) AS count,
		    countIf(ended_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM sessions
		WHERE organization_id = ?
		AND project_id = ?
		AND ended_at > now() - %[2]s
		GROUP BY page
		HAVING count > 0
		ORDER BY count DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
