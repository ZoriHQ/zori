package cards

import (
	"fmt"
	"time"

	goclick "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/cockroachdb/errors"

	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics"
	"zori/services/analytics/filters"
)

const (
	DeviceTypeMobile  = "Mobile"
	DeviceTypeDesktop = "Desktop"
)

type TimelineCardData struct {
	TimeBucket       time.Time `json:"time_bucket" ch:"time_bucket"`
	NumDesktopVisits uint64    `json:"num_desktop_visits" ch:"desktop"`
	NumMobileVisits  uint64    `json:"num_mobile_visits" ch:"mobile"`
	NumUnknownVisits uint64    `json:"num_unknown_visits" ch:"unknown"`
	NumRevenue       uint64    `json:"num_revenue" ch:"amount_in_cents"`
}

type TimelineCardResponse struct {
	Precision   CardPrecision      `json:"precision"`
	Data        []TimelineCardData `json:"data"`
	TotalVisits uint64             `json:"total_visits"`
}

type TimelineCard struct {
	db *clickhouse.ClickhouseDB
}

func NewTimelineCard(db *clickhouse.ClickhouseDB) *TimelineCard {
	return &TimelineCard{
		db: db,
	}
}

func (c *TimelineCard) Fetch(ctx *ctx.Ctx, filter filters.SectionFilter) (*TimelineCardResponse, error) {
	query := c.buildTimelineQuery(filter.TimeRange.IntervalFunction)

	rows, err := c.db.Db().Query(
		ctx,
		query,
		filter.ProjectID,
		filter.TimeRange.Start,
		filter.ProjectID,
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

	return &TimelineCardResponse{
		Precision:   FilterToTimePrecision(filter),
		Data:        dataPoints,
		TotalVisits: calculateTotalVisits(dataPoints),
	}, nil
}

func (c *TimelineCard) buildTimelineQuery(intervalFunc string) string {
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
		SELECT
			%[1]s(client_timestamp_utc) AS time_bucket,
			countIf(device_type = '%[2]s') AS desktop,
			countIf(device_type = '%[3]s') AS mobile,
			countIf(device_type IS NULL OR (device_type != '%[2]s' AND device_type != '%[3]s')) AS unknown,
			COALESCE(r.amount_in_cents, 0) AS amount_in_cents
		FROM events e
		LEFT JOIN revenue r ON %[1]s(e.client_timestamp_utc) = r.time_bucket
		WHERE e.project_id = ?
			AND e.client_timestamp_utc >= ?
			AND e.client_timestamp_utc <= now()
		GROUP BY time_bucket, r.amount_in_cents
		ORDER BY time_bucket ASC`,
		intervalFunc,
		DeviceTypeDesktop,
		DeviceTypeMobile,
	)
}

func (c *TimelineCard) scanTimelineData(rows goclick.Rows, projectID string) ([]TimelineCardData, error) {
	var dataPoints []TimelineCardData

	for rows.Next() {
		var datapoint TimelineCardData
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

func (c *TimelineCard) wrapFetchError(err error, projectID string) error {
	return errors.WithTelemetry(
		errors.Wrap(err, analytics.NewTimelineFetchError().Error()),
		"project_id", projectID,
	)
}

func calculateTotalVisits(dataPoints []TimelineCardData) uint64 {
	var total uint64
	for _, dp := range dataPoints {
		total += dp.NumDesktopVisits + dp.NumMobileVisits + dp.NumUnknownVisits
	}
	return total
}
