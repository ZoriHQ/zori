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
	Timestamp time.Time `json:"timestamp"`
	Mobile    uint64    `json:"mobile"`
	Desktop   uint64    `json:"desktop"`
	Tablet    uint64    `json:"tablet,omitempty"`
	Unknown   uint64    `json:"unknown,omitempty"`
}

// UniqueVisitorsTimelineResponse represents unique visitors over time split by device
type UniqueVisitorsTimelineResponse struct {
	Data []UniqueVisitorsDataPoint `json:"data"`
}

// UniqueVisitorsDataPoint represents unique visitors at a point in time, split by device
type UniqueVisitorsDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Mobile    uint64    `json:"mobile"`
	Desktop   uint64    `json:"desktop"`
}

// VisitorsByOriginResponse represents unique visitors grouped by traffic origin
type VisitorsByOriginResponse struct {
	Data []OriginDataPoint `json:"data"`
}

// OriginDataPoint represents visitors from a specific origin
type OriginDataPoint struct {
	Origin         string `json:"origin"`
	UniqueVisitors uint64 `json:"unique_visitors"`
}

// VisitorsByCountryResponse represents unique visitors grouped by country
type VisitorsByCountryResponse struct {
	Data []CountryDataPoint `json:"data"`
}

// CountryDataPoint represents visitors from a specific country
type CountryDataPoint struct {
	CountryCode    string `json:"country_code"`
	UniqueVisitors uint64 `json:"unique_visitors"`
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

// TopVisitorsRequest represents a request for top visitors
type TopVisitorsRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	Limit     int       `query:"limit"`
}

// TopVisitorsResponse represents the most active visitors
type TopVisitorsResponse struct {
	Visitors []TopVisitor `json:"visitors"`
	Total    int          `json:"total"`
}

// TopVisitor represents a single active visitor
type TopVisitor struct {
	VisitorID          string    `json:"visitor_id"`
	EventCount         uint64    `json:"event_count"`
	LastSeen           time.Time `json:"last_seen"`
	FirstSeen          time.Time `json:"first_seen"`
	LocationCountryISO *string   `json:"location_country_iso,omitempty"`
	LocationCity       *string   `json:"location_city,omitempty"`
	DeviceType         *string   `json:"device_type,omitempty"`
	BrowserName        *string   `json:"browser_name,omitempty"`
}

// VisitorProfileResponse represents a single visitor's profile
type VisitorProfileResponse struct {
	VisitorID          string         `json:"visitor_id"`
	FirstSeen          time.Time      `json:"first_seen"`
	LastSeen           time.Time      `json:"last_seen"`
	TotalEvents        uint64         `json:"total_events"`
	FirstTrafficOrigin *string        `json:"first_traffic_origin,omitempty"`
	FirstReferrerURL   *string        `json:"first_referrer_url,omitempty"`
	LocationCountryISO *string        `json:"location_country_iso,omitempty"`
	LocationCity       *string        `json:"location_city,omitempty"`
	Events             []VisitorEvent `json:"events"`
	EventsOverTime     []EventsOverTimeDataPoint `json:"events_over_time"`
}

// VisitorEvent represents a single event in a visitor's history
type VisitorEvent struct {
	EventName          *string   `json:"event_name"`
	ClientTimestampUTC time.Time `json:"client_timestamp_utc"`
	PageURL            string    `json:"page_url"`
	PagePath           string    `json:"page_path"`
	ReferrerURL        string    `json:"referrer_url,omitempty"`
	DeviceType         *string   `json:"device_type,omitempty"`
	BrowserName        *string   `json:"browser_name,omitempty"`
}

// EventsOverTimeDataPoint represents event count at a point in time
type EventsOverTimeDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	EventCount uint64    `json:"event_count"`
}
