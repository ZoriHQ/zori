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
	TotalRevenue         int64   `json:"total_revenue"`          // Total revenue in smallest currency unit (cents)
	TotalPayments        uint64  `json:"total_payments"`         // Count of successful payments
	Currency             string  `json:"currency,omitempty"`

	// Customer metrics
	PayingCustomers      uint64  `json:"paying_customers"`       // Unique customers who paid
	ConversionRate       float64 `json:"conversion_rate"`        // % of visitors who paid

	// Average metrics
	AvgOrderValue        int64   `json:"avg_order_value"`        // Average payment amount
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"` // Average revenue per paying customer
	AvgRevenuePerSession int64   `json:"avg_revenue_per_session"` // Average revenue per session

	// Identified customers (have email/user_id)
	IdentifiedCustomers          uint64  `json:"identified_customers"`
	IdentifiedCustomerRevenue    int64   `json:"identified_customer_revenue"`
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
	Origin               string  `json:"origin"`
	TotalRevenue         int64   `json:"total_revenue"`         // Revenue in smallest currency unit (cents)
	RevenuePercentage    float64 `json:"revenue_percentage"`
	PayingCustomers      uint64  `json:"paying_customers"`
	UniqueVisitors       uint64  `json:"unique_visitors"`
	ConversionRate       float64 `json:"conversion_rate"`         // paying_customers / unique_visitors * 100
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"` // Average revenue per paying customer
	PaymentCount         uint64  `json:"payment_count"`
	Currency             string  `json:"currency,omitempty"`
}

// AttributionByUTMRequest represents a request for revenue attribution by UTM parameters
type AttributionByUTMRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	UTMType   string    `query:"utm_type"` // "source", "medium", or "campaign"
}

// AttributionByUTMResponse represents revenue attributed to UTM parameters
type AttributionByUTMResponse struct {
	Data []UTMAttributionDataPoint `json:"data"`
}

// UTMAttributionDataPoint represents revenue from a specific UTM parameter
type UTMAttributionDataPoint struct {
	UTMValue             string  `json:"utm_value"`
	TotalRevenue         int64   `json:"total_revenue"`         // Revenue in smallest currency unit (cents)
	RevenuePercentage    float64 `json:"revenue_percentage"`
	PayingCustomers      uint64  `json:"paying_customers"`
	UniqueVisitors       uint64  `json:"unique_visitors"`
	ConversionRate       float64 `json:"conversion_rate"`         // paying_customers / unique_visitors * 100
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"` // Average revenue per paying customer
	PaymentCount         uint64  `json:"payment_count"`
	Currency             string  `json:"currency,omitempty"`
}

// TimelineRequest represents a request for revenue over time
type TimelineRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// TimelineResponse represents revenue over time
type TimelineResponse struct {
	Data []TimelineDataPoint `json:"data"`
}

// TimelineDataPoint represents revenue at a specific time bucket
type TimelineDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalRevenue float64   `json:"total_revenue"` // Revenue in smallest currency unit (cents)
	PaymentCount uint64    `json:"payment_count"`
	Currency     string    `json:"currency,omitempty"`
}

// TopCustomersRequest represents a request for top revenue customers
type TopCustomersRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	Limit     int       `query:"limit"` // Default 50
}

// TopCustomersResponse represents the highest revenue customers
type TopCustomersResponse struct {
	Customers []TopCustomer `json:"customers"`
	Total     int           `json:"total"`
}

// TopCustomer represents a single customer with revenue data
type TopCustomer struct {
	VisitorID          string     `json:"visitor_id"`
	UserID             *string    `json:"user_id,omitempty"`
	ExternalID         *string    `json:"external_id,omitempty"`
	Email              *string    `json:"email,omitempty"`
	Name               *string    `json:"name,omitempty"`
	TotalRevenue       int64      `json:"total_revenue"`       // Total revenue in smallest currency unit (cents)
	PaymentCount       uint64     `json:"payment_count"`
	FirstPaymentDate   time.Time  `json:"first_payment_date"`
	LastPaymentDate    time.Time  `json:"last_payment_date"`
	AvgOrderValue      int64      `json:"avg_order_value"`
	Currency           string     `json:"currency,omitempty"`
	FirstTrafficOrigin *string    `json:"first_traffic_origin,omitempty"`
	LocationCountryISO *string    `json:"location_country_iso,omitempty"`
}

// CustomerProfileRequest represents a request for a customer's revenue profile
type CustomerProfileRequest struct {
	ProjectID string `query:"project_id" validate:"required"`
	VisitorID string `query:"visitor_id" validate:"required"`
}

// CustomerProfileResponse represents detailed revenue profile for a customer
type CustomerProfileResponse struct {
	// Identity
	VisitorID  string  `json:"visitor_id"`
	UserID     *string `json:"user_id,omitempty"`
	ExternalID *string `json:"external_id,omitempty"`
	Email      *string `json:"email,omitempty"`
	Name       *string `json:"name,omitempty"`

	// Revenue summary
	TotalRevenue       int64      `json:"total_revenue"`       // Total revenue in smallest currency unit (cents)
	PaymentCount       uint64     `json:"payment_count"`
	FirstPaymentDate   *time.Time `json:"first_payment_date,omitempty"`
	LastPaymentDate    *time.Time `json:"last_payment_date,omitempty"`
	AvgOrderValue      int64      `json:"avg_order_value"`
	Currency           string     `json:"currency,omitempty"`

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

// Payment represents a single payment transaction
type Payment struct {
	PaymentID        string    `json:"payment_id"`
	Amount           int64     `json:"amount"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	ProductName      string    `json:"product_name"`
	PaymentTimestamp time.Time `json:"payment_timestamp"`
	ProviderType     string    `json:"provider_type"`
}

// RevenueOverTimeDataPoint represents revenue at a point in time
type RevenueOverTimeDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalRevenue int64     `json:"total_revenue"`
	PaymentCount uint64    `json:"payment_count"`
}

// ConversionMetricsRequest represents a request for conversion metrics
type ConversionMetricsRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

// ConversionMetricsResponse represents conversion funnel and metrics
type ConversionMetricsResponse struct {
	// Funnel metrics
	TotalVisitors     uint64  `json:"total_visitors"`
	PayingCustomers   uint64  `json:"paying_customers"`
	ConversionRate    float64 `json:"conversion_rate"` // % of visitors who paid

	// Time to conversion
	AvgTimeToFirstPurchase float64 `json:"avg_time_to_first_purchase_hours"` // Hours from first visit to first purchase
	MedianTimeToFirstPurchase float64 `json:"median_time_to_first_purchase_hours"`

	// Customer value
	CustomerLifetimeValue float64 `json:"customer_lifetime_value"` // Average total revenue per paying customer

	// Repeat purchase metrics
	RepeatPurchaseRate float64 `json:"repeat_purchase_rate"` // % of customers who made 2+ purchases
	AvgPurchasesPerCustomer float64 `json:"avg_purchases_per_customer"`
}
