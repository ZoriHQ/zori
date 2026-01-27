package types

import (
	"time"
	"zori/services/analytics/filters"
)

type AttributionFilterMode string

const (
	FilterByPaymentDate     AttributionFilterMode = "payment_date"
	FilterByAcquisitionDate AttributionFilterMode = "acquisition_date"
)

type VisitorsByDeviceResponse struct {
	Data []VisitorDataPoint `json:"data"`
}

type VisitorDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Mobile    uint64    `json:"mobile"`
	Desktop   uint64    `json:"desktop"`
	Unknown   uint64    `json:"unknown,omitempty"`
}

type UniqueVisitorsTimelineResponse struct {
	Data []UniqueVisitorsDataPoint `json:"data"`
}

type UniqueVisitorsDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Mobile    uint64    `json:"mobile"`
	Desktop   uint64    `json:"desktop"`
}

type VisitorsByOriginResponse struct {
	Data []OriginDataPoint `json:"data"`
}

type OriginDataPoint struct {
	Origin         string  `json:"origin"`
	UniqueVisitors uint64  `json:"unique_visitors"`
	Percentage     float64 `json:"percentage"`
}

type VisitorsByCountryResponse struct {
	Data []CountryDataPoint `json:"data"`
}

type CountryDataPoint struct {
	CountryCode    string  `json:"country_code"`
	UniqueVisitors uint64  `json:"unique_visitors"`
	Percentage     float64 `json:"percentage"`
}

type RecentEventsRequest struct {
	filters.SectionFilter
	UserID        *string `json:"user_id" query:"user_id" example:"user_456"`
	ExternalID    *string `json:"external_id" query:"external_id" example:"ext_789"`
	TrafficOrigin *string `json:"traffic_origin" query:"traffic_origin" example:"google.com"`
	PagePath      *string `json:"page_path" query:"page_path" example:"/pricing"`
	EventName     *string `json:"event_name" query:"event_name" example:"page_view,click"`
}

type RecentEventsResponse struct {
	Events []RecentEvent `json:"events"`
	Total  uint64        `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type RecentEvent struct {
	EventName          *string   `json:"event_name"`
	VisitorID          string    `json:"visitor_id"`
	SessionID          string    `json:"session_id"`
	UserID             *string   `json:"user_id,omitempty"`
	ExternalID         *string   `json:"external_id,omitempty"`
	ClientTimestampUTC time.Time `json:"client_timestamp_utc"`
	PageURL            string    `json:"page_url"`
	PagePath           string    `json:"page_path"`
	Host               string    `json:"host"`
	ReferrerURL        string    `json:"referrer_url,omitempty"`
	ReferrerDomain     *string   `json:"referrer_domain,omitempty"`
	DeviceType         *string   `json:"device_type,omitempty"`
	BrowserName        *string   `json:"browser_name,omitempty"`
	OsName             *string   `json:"os_name,omitempty"`
	LocationCountryISO *string   `json:"location_country_iso,omitempty"`
	LocationCity       *string   `json:"location_city,omitempty"`
	LocationLatitude   *float64  `json:"location_latitude,omitempty"`
	LocationLongitude  *float64  `json:"location_longitude,omitempty"`

	// UTM parameters for campaign attribution
	UTMSource   *string `json:"utm_source,omitempty"`
	UTMMedium   *string `json:"utm_medium,omitempty"`
	UTMCampaign *string `json:"utm_campaign,omitempty"`

	// Custom properties sent with the event
	CustomProperties map[string]any `json:"custom_properties,omitempty"`

	ClickElementTag      *string  `json:"click_element_tag,omitempty"`
	ClickElementSelector *string  `json:"click_element_selector,omitempty"`
	ClickElementText     *string  `json:"click_element_text,omitempty"`
	ClickPositionX       *float64 `json:"click_position_x,omitempty"`
	ClickPositionY       *float64 `json:"click_position_y,omitempty"`
	ClickScreenWidth     *uint16  `json:"click_screen_width,omitempty"`
	ClickScreenHeight    *uint16  `json:"click_screen_height,omitempty"`

	ClickElementType     *string `json:"click_element_type,omitempty"`
	ClickElementCategory *string `json:"click_element_category,omitempty"`
	IsCTAClick           *bool   `json:"is_cta_click,omitempty"`
	LinkDestination      *string `json:"link_destination,omitempty"`
	IsExternalLink       *bool   `json:"is_external_link,omitempty"`
	IsDownloadLink       *bool   `json:"is_download_link,omitempty"`
}

type TopVisitorsResponse struct {
	Visitors []TopVisitor `json:"visitors"`
	Total    int          `json:"total"`
}

type TopVisitor struct {
	UserID     *string `json:"user_id,omitempty"`
	ExternalID *string `json:"external_id,omitempty"`
	Email      *string `json:"email,omitempty"`
	Name       *string `json:"name,omitempty"`

	VisitorIDs []string `json:"visitor_ids"`
	IsGrouped  bool     `json:"is_grouped"`

	EventCount uint64    `json:"event_count"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`

	LocationCountryISO *string `json:"location_country_iso,omitempty"`
	LocationCity       *string `json:"location_city,omitempty"`
	DeviceType         *string `json:"device_type,omitempty"`
	BrowserName        *string `json:"browser_name,omitempty"`
}

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
}

type VisitorEvent struct {
	EventName          *string   `json:"event_name"`
	ClientTimestampUTC time.Time `json:"client_timestamp_utc"`
	PageURL            string    `json:"page_url"`
	PagePath           string    `json:"page_path"`
	ReferrerURL        string    `json:"referrer_url,omitempty"`
	DeviceType         *string   `json:"device_type,omitempty"`
	BrowserName        *string   `json:"browser_name,omitempty"`

	ClickElementTag      *string  `json:"click_element_tag,omitempty"`
	ClickElementSelector *string  `json:"click_element_selector,omitempty"`
	ClickElementText     *string  `json:"click_element_text,omitempty"`
	ClickPositionX       *float64 `json:"click_position_x,omitempty"`
	ClickPositionY       *float64 `json:"click_position_y,omitempty"`
	ClickScreenWidth     *uint16  `json:"click_screen_width,omitempty"`
	ClickScreenHeight    *uint16  `json:"click_screen_height,omitempty"`

	ClickElementType     *string `json:"click_element_type,omitempty"`
	ClickElementCategory *string `json:"click_element_category,omitempty"`
	IsCTAClick           *bool   `json:"is_cta_click,omitempty"`
	LinkDestination      *string `json:"link_destination,omitempty"`
	IsExternalLink       *bool   `json:"is_external_link,omitempty"`
	IsDownloadLink       *bool   `json:"is_download_link,omitempty"`
}

type EventsOverTimeDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	EventCount uint64    `json:"event_count"`
}

type SessionMetricsResponse struct {
	AverageSessionDuration float64 `json:"average_session_duration_seconds"`
	AveragePagesPerSession float64 `json:"average_pages_per_session"`
	TotalSessions          uint64  `json:"total_sessions"`
}

type BounceRateResponse struct {
	OverallBounceRate float64                  `json:"overall_bounce_rate"`
	ByPage            []BounceRateByPageMetric `json:"by_page"`
}

type BounceRateByPageMetric struct {
	Page       string  `json:"page"`
	Sessions   uint64  `json:"sessions"`
	BounceRate float64 `json:"bounce_rate"`
}

type ReturnRateResponse struct {
	ReturnRatePercent      float64 `json:"return_rate_percent"`
	TotalUsers             uint64  `json:"total_users"`
	ReturningUsers         uint64  `json:"returning_users"`
	AvgTimeBetweenSessions float64 `json:"avg_time_between_sessions_hours,omitempty"`
}

type ChurnRateResponse struct {
	ChurnRatePercent   float64 `json:"churn_rate_percent"`
	TotalUsers         uint64  `json:"total_users"`
	ChurnedUsers       uint64  `json:"churned_users"`
	ChurnThresholdDays int     `json:"churn_threshold_days"`
}

type CohortAnalysisResponse struct {
	Cohorts []CohortData `json:"cohorts"`
}

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

type ManualIdentifyResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	VisitorID string `json:"visitor_id"`
}

type EventFilterOptionsResponse struct {
	TrafficOrigins []string `json:"traffic_origins"`
	Pages          []string `json:"pages"`
	EventNames     []string `json:"event_names"`
}

