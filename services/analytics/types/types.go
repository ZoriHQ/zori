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

// AttributionFilterMode determines how to filter revenue attribution data
type AttributionFilterMode string

const (
	// FilterByPaymentDate filters by when payments occurred (recommended for revenue reports)
	FilterByPaymentDate AttributionFilterMode = "payment_date"
	// FilterByAcquisitionDate filters by when visitors first arrived (recommended for cohort analysis)
	FilterByAcquisitionDate AttributionFilterMode = "acquisition_date"
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
	Origin               string  `json:"origin"`
	UniqueVisitors       uint64  `json:"unique_visitors"`
	Percentage           float64 `json:"percentage"`
	TotalRevenue         *int64  `json:"total_revenue"` // Revenue in smallest currency unit (cents)
	RevenuePercentage    float64 `json:"revenue_percentage"`
	PayingVisitors       uint64  `json:"paying_visitors"`
	ConversionRate       float64 `json:"conversion_rate"`         // paying_visitors / unique_visitors * 100
	AvgRevenuePerVisitor float64 `json:"avg_revenue_per_visitor"` // Average revenue per paying visitor
	PaymentCount         uint64  `json:"payment_count"`
	Currency             string  `json:"currency,omitempty"`
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
	UserID             *string   `json:"user_id,omitempty"`
	ExternalID         *string   `json:"external_id,omitempty"`
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
	UserID             *string   `json:"user_id,omitempty"`
	ExternalID         *string   `json:"external_id,omitempty"`
	EventCount         uint64    `json:"event_count"`
	LastSeen           time.Time `json:"last_seen"`
	FirstSeen          time.Time `json:"first_seen"`
	LocationCountryISO *string   `json:"location_country_iso,omitempty"`
	LocationCity       *string   `json:"location_city,omitempty"`
	DeviceType         *string   `json:"device_type,omitempty"`
	BrowserName        *string   `json:"browser_name,omitempty"`
	TotalRevenue       *int64    `json:"total_revenue"` // Total revenue in smallest currency unit (cents)
	PaymentCount       uint64    `json:"payment_count"`
	Currency           string    `json:"currency,omitempty"`
}

// VisitorProfileResponse represents a single visitor's profile
type VisitorProfileResponse struct {
	VisitorID          string                    `json:"visitor_id"`
	UserID             *string                   `json:"user_id,omitempty"`
	ExternalID         *string                   `json:"external_id,omitempty"`
	IsIdentified       bool                      `json:"is_identified"`
	Email              *string                   `json:"email,omitempty"`
	Name               *string                   `json:"name,omitempty"`
	Phone              *string                   `json:"phone,omitempty"`
	CustomTraits       map[string]interface{}    `json:"custom_traits,omitempty"`
	FirstIdentifiedAt  *time.Time                `json:"first_identified_at,omitempty"`
	LastIdentifiedAt   *time.Time                `json:"last_identified_at,omitempty"`
	FirstSeen          time.Time                 `json:"first_seen"`
	LastSeen           time.Time                 `json:"last_seen"`
	TotalEvents        uint64                    `json:"total_events"`
	FirstTrafficOrigin *string                   `json:"first_traffic_origin,omitempty"`
	FirstReferrerURL   *string                   `json:"first_referrer_url,omitempty"`
	LocationCountryISO *string                   `json:"location_country_iso,omitempty"`
	LocationCity       *string                   `json:"location_city,omitempty"`
	Events             []VisitorEvent            `json:"events"`
	EventsOverTime     []EventsOverTimeDataPoint `json:"events_over_time"`
	// Revenue fields
	TotalRevenue     float64          `json:"total_revenue"` // Total revenue in smallest currency unit (cents)
	PaymentCount     uint64           `json:"payment_count"`
	FirstPaymentDate *time.Time       `json:"first_payment_date,omitempty"`
	LastPaymentDate  *time.Time       `json:"last_payment_date,omitempty"`
	AvgOrderValue    int64            `json:"avg_order_value"` // Average payment amount
	Currency         string           `json:"currency,omitempty"`
	Payments         []VisitorPayment `json:"payments,omitempty"`
}

// VisitorPayment represents a payment made by a visitor
type VisitorPayment struct {
	PaymentID        string    `json:"payment_id"`
	Amount           int64     `json:"amount"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	ProductName      string    `json:"product_name"`
	PaymentTimestamp time.Time `json:"payment_timestamp"`
	ProviderType     string    `json:"provider_type"`
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
	Page       string  `json:"page"`
	Sessions   uint64  `json:"sessions"`
	BounceRate float64 `json:"bounce_rate"`
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
	ChurnRatePercent   float64 `json:"churn_rate_percent"`
	TotalUsers         uint64  `json:"total_users"`
	ChurnedUsers       uint64  `json:"churned_users"`
	ChurnThresholdDays int     `json:"churn_threshold_days"`
}

// CohortAnalysisResponse represents cohort retention analysis
type CohortAnalysisResponse struct {
	Cohorts []CohortData `json:"cohorts"`
}

// CohortData represents a single cohort's retention data
type CohortData struct {
	CohortPeriod    time.Time `json:"cohort_period"`
	CohortSize      uint64    `json:"cohort_size"`
	Week1Retention  float64   `json:"week_1_retention"`
	Week2Retention  float64   `json:"week_2_retention"`
	Week4Retention  float64   `json:"week_4_retention"`
	Month1Retention float64   `json:"month_1_retention,omitempty"`
	Month2Retention float64   `json:"month_2_retention,omitempty"`
	Month3Retention float64   `json:"month_3_retention,omitempty"`
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
	SessionsToday         uint64  `json:"sessions_today"`
	TotalSessionsInPeriod uint64  `json:"total_sessions_in_period"`
	AvgSessionDuration    float64 `json:"avg_session_duration_seconds"`
	AvgPagesPerSession    float64 `json:"avg_pages_per_session"`

	// Engagement metrics
	BounceRate float64 `json:"bounce_rate"`
	ReturnRate float64 `json:"return_rate"`

	// Total metrics
	TotalEvents    uint64 `json:"total_events"`
	UniqueVisitors uint64 `json:"unique_visitors"`
	UniqueSessions uint64 `json:"unique_sessions"` // Total unique sessions in period

	// Revenue metrics - 4 key metrics as separate queries
	TotalRevenue                    int64   `json:"total_revenue"`                      // 1. Total revenue for all payments in period
	TotalRevenueIdentifiedCustomers int64   `json:"total_revenue_identified_customers"` // 2. Total revenue for identified customers only
	AvgRevenuePerSession            int64   `json:"avg_revenue_per_session"`            // 3. Average revenue per unique session
	ConversionRate                  float64 `json:"conversion_rate"`                    // 4. % of unique visitors who made payment

	// Additional revenue metrics (kept for compatibility)
	PayingVisitors                  uint64  `json:"paying_visitors"`                     // Number of unique visitors who paid
	ConversionToPaying              float64 `json:"conversion_to_paying"`                // Same as ConversionRate (deprecated, use ConversionRate)
	AvgRevenuePerVisitor            float64 `json:"avg_revenue_per_visitor"`             // Average revenue per paying visitor
	AvgRevenuePerIdentifiedCustomer float64 `json:"avg_revenue_per_identified_customer"` // Average revenue for identified visitors only (deprecated)
	TotalPayments                   uint64  `json:"total_payments"`                      // Count of successful payments
	Currency                        string  `json:"currency,omitempty"`
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

// ManualIdentifyRequest represents a manual identification request from the dashboard
type ManualIdentifyRequest struct {
	ProjectID            string                 `json:"project_id" validate:"required"`
	VisitorID            string                 `json:"visitor_id" validate:"required"`
	UserID               *string                `json:"user_id,omitempty"`
	ExternalID           *string                `json:"external_id,omitempty"`
	Email                *string                `json:"email,omitempty"`
	Name                 *string                `json:"name,omitempty"`
	Phone                *string                `json:"phone,omitempty"`
	AdditionalProperties map[string]interface{} `json:"additional_properties,omitempty"`
}

// ManualIdentifyResponse represents the response after manual identification
type ManualIdentifyResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	VisitorID string `json:"visitor_id"`
}

// RevenueByUTMRequest represents a request for revenue by UTM parameters
type RevenueByUTMRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	UTMType   string    `query:"utm_type"` // "source", "medium", or "campaign"
}

// RevenueByUTMResponse represents revenue grouped by UTM parameters
type RevenueByUTMResponse struct {
	Data []UTMRevenueDataPoint `json:"data"`
}

// UTMRevenueDataPoint represents revenue from a specific UTM parameter
type UTMRevenueDataPoint struct {
	UTMValue             string  `json:"utm_value"`     // The UTM parameter value
	TotalRevenue         int64   `json:"total_revenue"` // Revenue in smallest currency unit (cents)
	RevenuePercentage    float64 `json:"revenue_percentage"`
	PayingVisitors       uint64  `json:"paying_visitors"`
	UniqueVisitors       uint64  `json:"unique_visitors"`
	ConversionRate       float64 `json:"conversion_rate"`         // paying_visitors / unique_visitors * 100
	AvgRevenuePerVisitor float64 `json:"avg_revenue_per_visitor"` // Average revenue per paying visitor
	PaymentCount         uint64  `json:"payment_count"`
	Currency             string  `json:"currency,omitempty"`
}

// RevenueTimelineRequest represents a request for revenue over time
type RevenueTimelineRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// RevenueTimelineResponse represents revenue over time
type RevenueTimelineResponse struct {
	Data []RevenueTimelineDataPoint `json:"data"`
}

// RevenueTimelineDataPoint represents revenue at a specific time bucket
type RevenueTimelineDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalRevenue *float64  `json:"total_revenue"` // Revenue in smallest currency unit (cents)
	PaymentCount uint64    `json:"payment_count"`
	Currency     string    `json:"currency,omitempty"`
}
