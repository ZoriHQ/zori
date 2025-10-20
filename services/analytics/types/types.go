package types

import "time"

// TimeRange represents the time period for analytics queries
type TimeRange string

const (
	TimeRangeLastHour   TimeRange = "last_hour"
	TimeRangeToday      TimeRange = "today"
	TimeRangeLast7Days  TimeRange = "last_7_days"
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
	Origin         string  `json:"origin"`
	UniqueVisitors uint64  `json:"unique_visitors"`
	Percentage     float64 `json:"percentage"`
}

// VisitorsByCountryResponse represents unique visitors grouped by country
type VisitorsByCountryResponse struct {
	Data []CountryDataPoint `json:"data"`
}

// CountryDataPoint represents visitors from a specific country
type CountryDataPoint struct {
	CountryCode    string  `json:"country_code"`
	UniqueVisitors uint64  `json:"unique_visitors"`
	Percentage     float64 `json:"percentage"`
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
	VisitorID          string                    `json:"visitor_id"`
	FirstSeen          time.Time                 `json:"first_seen"`
	LastSeen           time.Time                 `json:"last_seen"`
	TotalEvents        uint64                    `json:"total_events"`
	FirstTrafficOrigin *string                   `json:"first_traffic_origin,omitempty"`
	FirstReferrerURL   *string                   `json:"first_referrer_url,omitempty"`
	LocationCountryISO *string                   `json:"location_country_iso,omitempty"`
	LocationCity       *string                   `json:"location_city,omitempty"`
	Events             []VisitorEvent            `json:"events"`
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

// SessionMetricsResponse represents session metrics for a time range
type SessionMetricsResponse struct {
	AverageSessionDuration float64 `json:"average_session_duration_seconds"`
	AveragePagesPerSession float64 `json:"average_pages_per_session"`
	TotalSessions          uint64  `json:"total_sessions"`
}

// BounceRateResponse represents bounce rate metrics
type BounceRateResponse struct {
	OverallBounceRate float64                  `json:"overall_bounce_rate"`
	ByPage            []BounceRateByPageMetric `json:"by_page"`
}

// BounceRateByPageMetric represents bounce rate for a specific page
type BounceRateByPageMetric struct {
	Page        string  `json:"page"`
	Sessions    uint64  `json:"sessions"`
	BounceRate  float64 `json:"bounce_rate"`
}

// ReturnRateResponse represents return rate metrics
type ReturnRateResponse struct {
	ReturnRatePercent      float64 `json:"return_rate_percent"`
	TotalUsers             uint64  `json:"total_users"`
	ReturningUsers         uint64  `json:"returning_users"`
	AvgTimeBetweenSessions float64 `json:"avg_time_between_sessions_hours,omitempty"`
}

// ChurnRateResponse represents churn rate metrics
type ChurnRateResponse struct {
	ChurnRatePercent      float64 `json:"churn_rate_percent"`
	TotalUsers            uint64  `json:"total_users"`
	ChurnedUsers          uint64  `json:"churned_users"`
	ChurnThresholdDays    int     `json:"churn_threshold_days"`
}

// CohortAnalysisResponse represents cohort retention analysis
type CohortAnalysisResponse struct {
	Cohorts []CohortData `json:"cohorts"`
}

// CohortData represents a single cohort's retention data
type CohortData struct {
	CohortPeriod      time.Time `json:"cohort_period"`
	CohortSize        uint64    `json:"cohort_size"`
	Week1Retention    float64   `json:"week_1_retention"`
	Week2Retention    float64   `json:"week_2_retention"`
	Week4Retention    float64   `json:"week_4_retention"`
	Month1Retention   float64   `json:"month_1_retention,omitempty"`
	Month2Retention   float64   `json:"month_2_retention,omitempty"`
	Month3Retention   float64   `json:"month_3_retention,omitempty"`
}

// ActiveUsersResponse represents DAU/WAU/MAU metrics
type ActiveUsersResponse struct {
	DAU uint64 `json:"dau"`
	WAU uint64 `json:"wau"`
	MAU uint64 `json:"mau"`
}

// DashboardMetricsResponse represents combined key metrics for a dashboard
type DashboardMetricsResponse struct {
	// Active users
	DAU uint64 `json:"dau"`
	WAU uint64 `json:"wau"`
	MAU uint64 `json:"mau"`

	// Sessions
	SessionsToday          uint64  `json:"sessions_today"`
	TotalSessionsInPeriod  uint64  `json:"total_sessions_in_period"`
	AvgSessionDuration     float64 `json:"avg_session_duration_seconds"`
	AvgPagesPerSession     float64 `json:"avg_pages_per_session"`

	// Engagement metrics
	BounceRate             float64 `json:"bounce_rate"`
	ReturnRate             float64 `json:"return_rate"`

	// Total metrics
	TotalEvents            uint64  `json:"total_events"`
	UniqueVisitors         uint64  `json:"unique_visitors"`
}

// SessionMetricsRequest represents a request for session metrics
type SessionMetricsRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// BounceRateRequest represents a request for bounce rate
type BounceRateRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	Limit     int       `query:"limit"` // For by-page breakdown
}

// ReturnRateRequest represents a request for return rate
type ReturnRateRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// ChurnRateRequest represents a request for churn rate
type ChurnRateRequest struct {
	ProjectID          string    `query:"project_id" validate:"required"`
	TimeRange          TimeRange `query:"time_range" validate:"required"`
	ChurnThresholdDays int       `query:"churn_threshold_days"` // Default 30
}

// CohortAnalysisRequest represents a request for cohort analysis
type CohortAnalysisRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// ActiveUsersRequest represents a request for active users
type ActiveUsersRequest struct {
	ProjectID string `query:"project_id" validate:"required"`
}

// DashboardMetricsRequest represents a request for dashboard metrics
type DashboardMetricsRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}
