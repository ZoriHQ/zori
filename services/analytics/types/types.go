package types

import "time"

// TimeRange represents the time period for analytics queries
type TimeRange string

const (
	TimeRangeLastHour  TimeRange = "last_hour"
	TimeRangeToday     TimeRange = "today"
	TimeRangeLast7Days TimeRange = "last_7_days"
	TimeRangeLast30Days TimeRange = "last_30_days"
	TimeRangeLast90Days TimeRange = "last_90_days"
)

// VisitorsRequest represents a request for visitors analytics
type VisitorsRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// VisitorsByDeviceResponse represents visitors grouped by device type
type VisitorsByDeviceResponse struct {
	Data []VisitorDataPoint `json:"data"`
}

// VisitorDataPoint represents a single data point in time series
type VisitorDataPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Mobile    int               `json:"mobile"`
	Desktop   int               `json:"desktop"`
	Tablet    int               `json:"tablet,omitempty"`
	Unknown   int               `json:"unknown,omitempty"`
}

// VisitorsByOriginResponse represents unique visitors grouped by traffic origin
type VisitorsByOriginResponse struct {
	Data []OriginDataPoint `json:"data"`
}

// OriginDataPoint represents visitors from a specific origin
type OriginDataPoint struct {
	Origin         string `json:"origin"`
	UniqueVisitors int    `json:"unique_visitors"`
}

// VisitorsByCountryResponse represents unique visitors grouped by country
type VisitorsByCountryResponse struct {
	Data []CountryDataPoint `json:"data"`
}

// CountryDataPoint represents visitors from a specific country
type CountryDataPoint struct {
	CountryCode    string `json:"country_code"`
	UniqueVisitors int    `json:"unique_visitors"`
}

// RecentEventsResponse represents the most recent events
type RecentEventsResponse struct {
	Events []RecentEvent `json:"events"`
	Total  int           `json:"total"`
}

// RecentEvent represents a single recent event
type RecentEvent struct {
	EventName          *string   `json:"event_name"`
	VisitorID          string    `json:"visitor_id"`
	ClientTimestampUTC time.Time `json:"client_timestamp_utc"`
	PageURL            string    `json:"page_url"`
	PagePath           string    `json:"page_path"`
	ReferrerURL        string    `json:"referrer_url,omitempty"`
	DeviceType         *string   `json:"device_type,omitempty"`
	BrowserName        *string   `json:"browser_name,omitempty"`
	LocationCountryISO *string   `json:"location_country_iso,omitempty"`
	LocationCity       *string   `json:"location_city,omitempty"`
}
