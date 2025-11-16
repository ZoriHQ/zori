package tiles

import (
	"fmt"
	"time"

	goclick "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/cockroachdb/errors"

	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"
)

const (
	DeviceTypeMobile  = "Mobile"
	DeviceTypeDesktop = "Desktop"
)

type TimelineTileData struct {
	TimeBucket       time.Time `json:"time_bucket" ch:"time_bucket"`
	NumDesktopVisits uint64    `json:"num_desktop_visits" ch:"desktop"`
	NumMobileVisits  uint64    `json:"num_mobile_visits" ch:"mobile"`
	NumUnknownVisits uint64    `json:"num_unknown_visits" ch:"unknown"`
	NumRevenue       *int64    `json:"num_revenue" ch:"amount_in_cents"`
}

type TimelineTileResponse struct {
	Precision   CardPrecision      `json:"precision"`
	Data        []TimelineTileData `json:"data"`
	TotalVisits uint64             `json:"total_visits"`
}

type TimelineTile struct {
	db *clickhouse.ClickhouseDB
}

func NewTimelineTile(db *clickhouse.ClickhouseDB) *TimelineTile {
	return &TimelineTile{
		db: db,
	}
}

func (c *TimelineTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*TimelineTileResponse, error) {
	query := c.buildTimelineQuery(filter)

	rows, err := c.db.Db().Query(
		ctx,
		query,
		filter.ProjectID,
		filter.TimeRange.Start,
		filter.ProjectID,
		filter.TimeRange.Start,
		filter.TimeRange.Start,
	)

	if err != nil {
		return nil, c.wrapFetchError(err, filter.ProjectID)
	}
	defer clickhouse.EnsureClosed(rows)

	dataPoints, err := c.scanTimelineData(rows, filter.ProjectID)
	if err != nil {
		return nil, err
	}

	return &TimelineTileResponse{
		Precision:   FilterToTimePrecision(filter),
		Data:        dataPoints,
		TotalVisits: calculateTotalVisits(dataPoints),
	}, nil
}

func (c *TimelineTile) buildTimelineQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		WITH revenue AS (
			SELECT
				%[1]s(created_at) AS time_bucket,
				sum(amount) AS amount_in_cents
			FROM payment_events
			WHERE project_id = ?
				AND created_at >= ?
				AND created_at <= now()
			GROUP BY time_bucket
		)
		SELECT %[1]s(toDateTime(client_timestamp_utc))                                                             AS time_bucket,
		countDistinctIf(visitor_id, device_type = 'Desktop')                                                       AS desktop,
	    countDistinctIf(visitor_id, device_type = 'Mobile')                                                        AS mobile,
	    countDistinctIf(visitor_id, device_type IS NULL OR (device_type != 'Desktop' AND device_type != 'Mobile')) AS unknown,
		COALESCE(r.amount_in_cents, 0) AS amount_in_cents
		FROM events e
		LEFT JOIN revenue r ON %[1]s(e.client_timestamp_utc) = r.time_bucket
		WHERE e.project_id = ?
		  AND e.client_timestamp_utc >= ?
		  AND e.client_timestamp_utc <= now()
		GROUP BY ALL
		ORDER BY time_bucket ASC
		WITH FILL FROM %[1]s(toDateTime(?)) TO %[1]s(now()) STEP %[5]s(1);`,
		filter.TimeRange.TimeBucketFunction,
		DeviceTypeDesktop,
		DeviceTypeMobile,
		filter.TimeRange.IntervalValue,
		filter.TimeRange.ToIntervalStepFunction,
	)
}

func (c *TimelineTile) scanTimelineData(rows goclick.Rows, projectID string) ([]TimelineTileData, error) {
	var dataPoints []TimelineTileData

	for rows.Next() {
		var datapoint TimelineTileData
		if err := rows.ScanStruct(&datapoint); err != nil {
			return nil, c.wrapFetchError(err, projectID)
		}
		dataPoints = append(dataPoints, datapoint)
	}

	if err := rows.Err(); err != nil {
		return nil, c.wrapFetchError(err, projectID)
	}

	return dataPoints, nil
}

func (c *TimelineTile) wrapFetchError(err error, projectID string) error {
	return errors.WithTelemetry(
		errors.Wrap(err, NewTimelineFetchError().Error()),
		"project_id", projectID,
	)
}
