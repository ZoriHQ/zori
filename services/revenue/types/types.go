package types

import "time"

// TimeRange represents the time period for revenue queries
type TimeRange string

const (
	TimeRangeLastHour   TimeRange = "last_hour"
	TimeRangeToday      TimeRange = "today"
	TimeRangeLast7Days  TimeRange = "last_7_days"
	TimeRangeLast30Days TimeRange = "last_30_days"
	TimeRangeLast90Days TimeRange = "last_90_days"
)

// DashboardRequest represents a request for revenue dashboard metrics
type DashboardRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// DashboardResponse represents key revenue metrics for a dashboard
type DashboardResponse struct {
	// Core revenue metrics
	TotalRevenue  int64  `json:"total_revenue"`  // Total revenue in smallest currency unit (cents)
	TotalPayments uint64 `json:"total_payments"` // Count of successful payments
	Currency      string `json:"currency,omitempty"`

	// Customer metrics
	PayingCustomers uint64  `json:"paying_customers"` // Unique customers who paid
	ConversionRate  float64 `json:"conversion_rate"`  // % of visitors who paid

	// Average metrics
	AvgOrderValue         float64 `json:"avg_order_value"`          // Average payment amount
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"` // Average revenue per paying customer
	AvgRevenuePerSession  float64 `json:"avg_revenue_per_session"`  // Average revenue per session

	// Identified customers (have email/user_id)
	IdentifiedCustomers             uint64  `json:"identified_customers"`
	IdentifiedCustomerRevenue       int64   `json:"identified_customer_revenue"`
	AvgRevenuePerIdentifiedCustomer float64 `json:"avg_revenue_per_identified_customer"`
}

// AttributionByOriginRequest represents a request for revenue attribution by traffic origin
type AttributionByOriginRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// AttributionByOriginResponse represents revenue attributed to traffic origins
type AttributionByOriginResponse struct {
	Data []OriginAttributionDataPoint `json:"data"`
}

// OriginAttributionDataPoint represents revenue from a specific traffic origin
type OriginAttributionDataPoint struct {
	Origin                string  `json:"origin"`
	TotalRevenue          int64   `json:"total_revenue"` // Revenue in smallest currency unit (cents)
	RevenuePercentage     float64 `json:"revenue_percentage"`
	PayingCustomers       uint64  `json:"paying_customers"`
	UniqueVisitors        uint64  `json:"unique_visitors"`
	ConversionRate        float64 `json:"conversion_rate"`          // paying_customers / unique_visitors * 100
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"` // Average revenue per paying customer
	PaymentCount          uint64  `json:"payment_count"`
	Currency              string  `json:"currency,omitempty"`
}

type AttributionByUTMRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	UTMType   string    `query:"utm_type"` // "source", "medium", or "campaign"
}

type AttributionByUTMResponse struct {
	Data []UTMAttributionDataPoint `json:"data"`
}

type UTMAttributionDataPoint struct {
	UTMValue              string  `json:"utm_value"`
	TotalRevenue          int64   `json:"total_revenue"` // Revenue in smallest currency unit (cents)
	RevenuePercentage     float64 `json:"revenue_percentage"`
	PayingCustomers       uint64  `json:"paying_customers"`
	UniqueVisitors        uint64  `json:"unique_visitors"`
	ConversionRate        float64 `json:"conversion_rate"`          // paying_customers / unique_visitors * 100
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"` // Average revenue per paying customer
	PaymentCount          uint64  `json:"payment_count"`
	Currency              string  `json:"currency,omitempty"`
}

type TimelineRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

type TimelineResponse struct {
	Data []TimelineDataPoint `json:"data"`
}

type TimelineDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalRevenue int64     `json:"total_revenue"` // Revenue in smallest currency unit (cents)
	PaymentCount uint64    `json:"payment_count"`
	Currency     string    `json:"currency,omitempty"`
}

type TopCustomersRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	Limit     int       `query:"limit"` // Default 50
}

type TopCustomersResponse struct {
	Customers []TopCustomer `json:"customers"`
	Total     int           `json:"total"`
}

type TopCustomer struct {
	VisitorID          string    `json:"visitor_id"`
	UserID             *string   `json:"user_id,omitempty"`
	ExternalID         *string   `json:"external_id,omitempty"`
	Email              *string   `json:"email,omitempty"`
	Name               *string   `json:"name,omitempty"`
	TotalRevenue       int64     `json:"total_revenue"` // Total revenue in smallest currency unit (cents)
	PaymentCount       uint64    `json:"payment_count"`
	FirstPaymentDate   time.Time `json:"first_payment_date"`
	LastPaymentDate    time.Time `json:"last_payment_date"`
	AvgOrderValue      float64   `json:"avg_order_value"`
	Currency           string    `json:"currency,omitempty"`
	FirstTrafficOrigin *string   `json:"first_traffic_origin,omitempty"`
	LocationCountryISO *string   `json:"location_country_iso,omitempty"`
}

type CustomerProfileRequest struct {
	ProjectID string `query:"project_id" validate:"required"`
	VisitorID string `query:"visitor_id" validate:"required"`
}

type CustomerProfileResponse struct {
	// Identity
	VisitorID  string  `json:"visitor_id"`
	UserID     *string `json:"user_id,omitempty"`
	ExternalID *string `json:"external_id,omitempty"`
	Email      *string `json:"email,omitempty"`
	Name       *string `json:"name,omitempty"`

	// Revenue summary
	TotalRevenue     int64      `json:"total_revenue"` // Total revenue in smallest currency unit (cents)
	PaymentCount     uint64     `json:"payment_count"`
	FirstPaymentDate *time.Time `json:"first_payment_date,omitempty"`
	LastPaymentDate  *time.Time `json:"last_payment_date,omitempty"`
	AvgOrderValue    int64      `json:"avg_order_value"`
	Currency         string     `json:"currency,omitempty"`

	// Attribution
	FirstTrafficOrigin *string `json:"first_traffic_origin,omitempty"`
	FirstUTMSource     *string `json:"first_utm_source,omitempty"`
	FirstUTMMedium     *string `json:"first_utm_medium,omitempty"`
	FirstUTMCampaign   *string `json:"first_utm_campaign,omitempty"`

	// Payment history
	Payments []Payment `json:"payments"`

	// Revenue over time (last 90 days)
	RevenueOverTime []RevenueOverTimeDataPoint `json:"revenue_over_time"`
}

type Payment struct {
	PaymentID        string    `json:"payment_id"`
	Amount           int64     `json:"amount"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	ProductName      string    `json:"product_name"`
	PaymentTimestamp time.Time `json:"payment_timestamp"`
	ProviderType     string    `json:"provider_type"`
}

type RevenueOverTimeDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalRevenue int64     `json:"total_revenue"`
	PaymentCount uint64    `json:"payment_count"`
}

type ConversionMetricsRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

type ConversionMetricsResponse struct {
	TotalVisitors   uint64  `json:"total_visitors"`
	PayingCustomers uint64  `json:"paying_customers"`
	ConversionRate  float64 `json:"conversion_rate"` // % of visitors who paid

	// Time to conversion
	AvgTimeToFirstPurchase    float64 `json:"avg_time_to_first_purchase_hours"` // Hours from first visit to first purchase
	MedianTimeToFirstPurchase float64 `json:"median_time_to_first_purchase_hours"`

	// Customer value
	CustomerLifetimeValue float64 `json:"customer_lifetime_value"` // Average total revenue per paying customer

	// Repeat purchase metrics
	RepeatPurchaseRate      float64 `json:"repeat_purchase_rate"` // % of customers who made 2+ purchases
	AvgPurchasesPerCustomer float64 `json:"avg_purchases_per_customer"`
}
