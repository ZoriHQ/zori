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
		WITH first_events AS (
		    SELECT
		        session_id,
		        argMin(ifNull(nullIf(page_pattern, ''), page_path), client_timestamp_utc) AS page,
		        min(client_timestamp_utc) AS timestamp
		    FROM events
		    WHERE organization_id = ?
		    AND project_id = ?
		    AND session_id != ''
		    AND client_timestamp_utc > now() - %[2]s
		    GROUP BY session_id
		)
		SELECT
		    ifNull(nullIf(page, ''), 'UNKNOWN') AS page,
		    countIf(timestamp > now() - %[1]s) AS count,
		    countIf(timestamp BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM first_events
		GROUP BY page
		HAVING count > 0
		ORDER BY count DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
