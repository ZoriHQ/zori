package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

type VisitorsByOSData struct {
	BrowserName   string `json:"browser_name" ch:"browser_name"`
	Count         uint64 `json:"count" ch:"count"`
	PreviousCount uint64 `json:"previous_count" ch:"previous_count"`
}

type VisitorsByOSResponse struct {
	Data []*VisitorsByOSData `json:"data"`
}

type VisitorsByOSTile struct {
	db *clickhouse.ClickhouseDB
}

func NewVisitorsByOSTile(db *clickhouse.ClickhouseDB) *VisitorsByOSTile {
	return &VisitorsByOSTile{db: db}
}

func (t *VisitorsByOSTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*VisitorsByOSResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &VisitorsByOSResponse{Data: data}, nil
}

func (t *VisitorsByOSTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]*VisitorsByOSData, error) {
	query := t.buildQuery(filter)

	row, err := t.db.Db().Query(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)
	if err != nil {
		return nil, err
	}

	if row.Err() != nil {
		return nil, row.Err()
	}

	var visitorsByOS []*VisitorsByOSData
	for row.Next() {
		var data VisitorsByOSData
		err := row.ScanStruct(&data)
		if err != nil {
			return nil, err
		}
		visitorsByOS = append(visitorsByOS, &data)
	}

	return visitorsByOS, nil
}

func (t *VisitorsByOSTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
			ifNull(nullIf(os_name, ''), 'UNKNOWN') AS os_name,
			uniqIf(visitor_id, created_at > now() - %[1]s) AS count,
			uniqIf(visitor_id, created_at BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_count
		FROM events
		WHERE events.organization_id = ?
		AND events.project_id = ?
		GROUP BY os_name
		HAVING count > 0
		ORDER BY count DESC
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}
