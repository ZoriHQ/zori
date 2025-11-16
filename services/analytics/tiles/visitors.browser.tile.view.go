package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type VisitorsByBrowserData struct {
	BrowserName   string `json:"browser_name" ch:"browser_name"`
	Count         uint64 `json:"count" ch:"count"`
	PreviousCount uint64 `json:"previous_count" ch:"previous_count"`
}

type VisitorsByBrowserResponse struct {
	Data []*VisitorsByBrowserData `json:"data"`
}

type VisitorsByBrowserTile struct {
	db *clickhouse.ClickhouseDB
}

func NewVisitorsByBrowserTile(db *clickhouse.ClickhouseDB) *VisitorsByBrowserTile {
	return &VisitorsByBrowserTile{db: db}
}

func (t *VisitorsByBrowserTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*VisitorsByBrowserResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &VisitorsByBrowserResponse{Data: data}, nil
}

func (t *VisitorsByBrowserTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]*VisitorsByBrowserData, error) {
	query := t.buildQuery(filter)

	row, err := t.db.Db().Query(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)
	if err != nil {
		return nil, err
	}

	if row.Err() != nil {
		return nil, row.Err()
	}

	var visitorsByOS []*VisitorsByBrowserData
	for row.Next() {
		var data VisitorsByBrowserData
		err := row.ScanStruct(&data)
		if err != nil {
			return nil, err
		}
		visitorsByOS = append(visitorsByOS, &data)
	}

	return visitorsByOS, nil
}

func (t *VisitorsByBrowserTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
			ifNull(nullIf(browser_name, ''), 'UNKNOWN') AS browser_name,
			uniqIf(visitor_id, created_at > now() - %[1]s) AS count,
			uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM events
		WHERE events.organization_id = ?
		AND events.project_id = ?
		GROUP BY browser_name
		HAVING count > 0
		ORDER BY count DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
