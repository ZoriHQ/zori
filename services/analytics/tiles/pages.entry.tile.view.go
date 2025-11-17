package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type EntryPagesData struct {
	Page          string `json:"page" ch:"page"`
	Count         uint64 `json:"count" ch:"count"`
	PreviousCount uint64 `json:"previous_count" ch:"previous_count"`
}

type EntryPagesResponse struct {
	Data []*EntryPagesData `json:"data"`
}

type EntryPagesTile struct {
	db *clickhouse.ClickhouseDB
}

func NewEntryPagesTile(db *clickhouse.ClickhouseDB) *EntryPagesTile {
	return &EntryPagesTile{db: db}
}

func (t *EntryPagesTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*EntryPagesResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &EntryPagesResponse{Data: data}, nil
}

func (t *EntryPagesTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]*EntryPagesData, error) {
	query := t.buildQuery(filter)

	queryResult, err := t.db.Db().Query(ctx, query, ctx.OrgID(), filter.ProjectID)
	if err != nil {
		return nil, err
	}
	defer clickhouse.EnsureClosed(queryResult)

	var entryPagesData []*EntryPagesData
	for queryResult.Next() {
		var data EntryPagesData
		err := queryResult.ScanStruct(&data)
		if err != nil {
			return nil, err
		}
		entryPagesData = append(entryPagesData, &data)
	}

	return entryPagesData, nil
}

func (t *EntryPagesTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    ifNull(nullIf(entry_page, ''), 'UNKNOWN') AS page,
		    countIf(started_at > now() - %[1]s) AS count,
		    countIf(started_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM sessions
		WHERE organization_id = ?
		AND project_id = ?
		AND started_at > now() - %[2]s
		GROUP BY page
		HAVING count > 0
		ORDER BY count DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
