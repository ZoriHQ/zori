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
	ProjectID  string  `json:"project_id" query:"projectID" form:"project_id" validate:"required" example:"proj_123"`
	VisitorID  *string `json:"visitor_id" query:"visitorID" form:"visitor_id" validate:"omitempty" example:"visitor_abc123"`
	CustomerID *string `json:"customer_id" query:"customerID" form:"customer_id" validate:"omitempty" example:"cus_xyz789"`
}

type SectionFilter struct {
	ProjectID      string         `json:"project_id" query:"projectID" form:"project_id" validate:"required" example:"proj_123"`
	TimeBoundaries TimeBoundaries `json:"time_range" query:"timeRange" form:"time_range" validate:"required,oneof=last_hour today yesterday last_7_days last_30_days last_90_days" enums:"last_hour,today,yesterday,last_7_days,last_30_days,last_90_days" default:"last_7_days" example:"last_7_days"`
	Referrer       *string        `json:"referrer" query:"referrer" form:"referrer" validate:"omitempty" example:"https://google.com"`
	VisitorID      *string        `json:"visitor_id" query:"visitorID" form:"visitor_id" validate:"omitempty" example:"visitor_abc123"`
	CustomerID     *string        `json:"customer_id" query:"customerID" form:"customer_id" validate:"omitempty" example:"cus_xyz789"`

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
	Start            time.Time
	IntervalFunction string
}

func ValidateTimeRange(tb TimeBoundaries) (*TimeRange, error) {
	now := time.Now().UTC()

	switch tb {
	case TimeBoundariesHour:
		return &TimeRange{
			Start:            now.Add(-1 * time.Hour),
			IntervalFunction: "toStartOfMinute",
		}, nil
	case TimeBoundariesToday:
		return &TimeRange{
			Start:            time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
			IntervalFunction: "toStartOfHour",
		}, nil
	case TimeBoundariesLastWeek:
		return &TimeRange{
			Start:            now.AddDate(0, 0, -7),
			IntervalFunction: "toStartOfHour",
		}, nil
	case TimeBoundariesLastMonth:
		return &TimeRange{
			Start:            now.AddDate(0, 0, -30),
			IntervalFunction: "toStartOfDay",
		}, nil
	case TimeBoundariesLast90Days:
		return &TimeRange{
			Start:            now.AddDate(0, 0, -90),
			IntervalFunction: "toStartOfDay",
		}, nil
	default:
		return nil, fmt.Errorf("invalid time range: %s", tb)
	}
}
