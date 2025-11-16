package filters

import (
	"fmt"
	"time"
)

type TimeBoundaries string

const (
	TimeBoundariesHour       TimeBoundaries = "last_hour"
	TimeBoundariesToday      TimeBoundaries = "today"
	TimeBoundariesYesterday  TimeBoundaries = "yesterday"
	TimeBoundariesLastWeek   TimeBoundaries = "last_7_days"
	TimeBoundariesLastMonth  TimeBoundaries = "last_30_days"
	TimeBoundariesLast90Days TimeBoundaries = "last_90_days"
)

type VisitorProfileFilter struct {
	ProjectID  string  `json:"project_id" query:"project_id" form:"project_id" validate:"required" example:"proj_123"`
	VisitorID  *string `json:"visitor_id" query:"visitor_id" form:"visitor_id" validate:"omitempty" example:"visitor_abc123"`
	CustomerID *string `json:"customer_id" query:"customer_id" form:"customer_id" validate:"omitempty" example:"cus_xyz789"`
}

type SectionFilter struct {
	ProjectID      string         `json:"project_id" query:"project_id" form:"project_id" validate:"required" example:"proj_123"`
	TimeBoundaries TimeBoundaries `json:"time_range" query:"time_range" form:"time_range" validate:"required,oneof=last_hour today yesterday last_7_days last_30_days last_90_days" enums:"last_hour,today,yesterday,last_7_days,last_30_days,last_90_days" default:"last_7_days" example:"last_7_days"`
	Referrer       *string        `json:"referrer" query:"referrer" form:"referrer" validate:"omitempty" example:"https://google.com"`
	VisitorID      *string        `json:"visitor_id" query:"visitor_id" form:"visitor_id" validate:"omitempty" example:"visitor_abc123"`
	CustomerID     *string        `json:"customer_id" query:"customer_id" form:"customer_id" validate:"omitempty" example:"cus_xyz789"`

	Limit  int `json:"limit" query:"limit" form:"limit" validate:"omitempty,min=1,max=100" default:"50" minimum:"1" maximum:"100" example:"50"`
	Offset int `json:"offset" query:"offset" form:"offset" validate:"omitempty,min=0" default:"0" minimum:"0" example:"0"`

	UTMFilter
	// TimeRange is a computed field
	// It helps us to get a start time and interval function for clickhouse queries
	TimeRange *TimeRange `json:"-" swaggerignore:"true"`
}

type UTMFilter struct {
	UTMTag      *string `query:"utm_tag" validate:"omitempty" example:"utm_source"`
	UTMTagValue *string `query:"utm_tag_value" validate:"omitempty" example:"google"`
}

type TimeRange struct {
	// Start is a start time we want to count or data from
	Start time.Time
	// DeltaOffset is a offset twice as big as Start, but in the past
	// It's used to calculate a difference in data change within a given range
	// So if the start time is 1h ago, DeltaOffset will be 2h ago
	DeltaOffset time.Time
	// TimeBucketFunction is a function that will be used to bucketize time to specific granularity
	// based on the given time boundaries
	TimeBucketFunction string

	// IntervalValue will represent a interval value for ClickHouse queries
	// based on the given time boundaries
	IntervalValue string
	// IntervalValueDelta will represent a interval value for ClickHouse queries
	// based on the given time boundaries twice as big as IntervalValue
	// So if IntervalValue is INTERVAL - 1 HOUR, IntervalValueDelta will be INTERVAL - 2 HOUR
	IntervalValueDelta string

	ToIntervalStepFunction string
}

func ValidateTimeRange(tb TimeBoundaries) (*TimeRange, error) {
	now := time.Now().UTC()

	switch tb {
	case TimeBoundariesHour:
		return &TimeRange{
			Start:                  now.Add(-1 * time.Hour),
			DeltaOffset:            now.Add(-2 * time.Hour),
			IntervalValue:          "INTERVAL 1 HOUR",
			IntervalValueDelta:     "INTERVAL 2 HOUR",
			TimeBucketFunction:     "toStartOfMinute",
			ToIntervalStepFunction: "toIntervalMinute",
		}, nil
	case TimeBoundariesToday:
		return &TimeRange{
			Start:                  time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
			DeltaOffset:            time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour),
			IntervalValue:          "INTERVAL 1 DAY",
			IntervalValueDelta:     "INTERVAL 2 DAY",
			TimeBucketFunction:     "toStartOfHour",
			ToIntervalStepFunction: "toIntervalHour",
		}, nil
	case TimeBoundariesLastWeek:
		return &TimeRange{
			Start:                  now.AddDate(0, 0, -7),
			DeltaOffset:            now.AddDate(0, 0, -14),
			IntervalValue:          "INTERVAL 7 DAY",
			IntervalValueDelta:     "INTERVAL 14 DAY",
			TimeBucketFunction:     "toStartOfDay",
			ToIntervalStepFunction: "toIntervalDay",
		}, nil
	case TimeBoundariesLastMonth:
		return &TimeRange{
			Start:                  now.AddDate(0, 0, -30),
			DeltaOffset:            now.AddDate(0, 0, -60),
			IntervalValue:          "INTERVAL 30 DAY",
			IntervalValueDelta:     "INTERVAL 60 DAY",
			TimeBucketFunction:     "toStartOfDay",
			ToIntervalStepFunction: "toIntervalDay",
		}, nil
	case TimeBoundariesLast90Days:
		return &TimeRange{
			Start:                  now.AddDate(0, 0, -90),
			DeltaOffset:            now.AddDate(0, 0, -180),
			IntervalValue:          "INTERVAL 90 DAY",
			IntervalValueDelta:     "INTERVAL 180 DAY",
			TimeBucketFunction:     "toStartOfDay",
			ToIntervalStepFunction: "toIntervalDay",
		}, nil
	default:
		return nil, fmt.Errorf("invalid time range: %s", tb)
	}
}
