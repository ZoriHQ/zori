package types

import "time"

type TimeRange string

const (
	TimeRangeLastHour   TimeRange = "last_hour"
	TimeRangeToday      TimeRange = "today"
	TimeRangeLast7Days  TimeRange = "last_7_days"
	TimeRangeLast30Days TimeRange = "last_30_days"
	TimeRangeLast90Days TimeRange = "last_90_days"
)

type DashboardRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

type DashboardResponse struct {
	TotalRevenue  int64  `json:"total_revenue"`
	TotalPayments uint64 `json:"total_payments"`
	Currency      string `json:"currency,omitempty"`

	PayingCustomers uint64  `json:"paying_customers"`
	ConversionRate  float64 `json:"conversion_rate"`

	AvgOrderValue         float64 `json:"avg_order_value"`
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"`
	AvgRevenuePerSession  float64 `json:"avg_revenue_per_session"`

	IdentifiedCustomers             uint64  `json:"identified_customers"`
	IdentifiedCustomerRevenue       int64   `json:"identified_customer_revenue"`
	AvgRevenuePerIdentifiedCustomer float64 `json:"avg_revenue_per_identified_customer"`
}

type AttributionByOriginRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
}

type AttributionByOriginResponse struct {
	Data []OriginAttributionDataPoint `json:"data"`
}

type OriginAttributionDataPoint struct {
	Origin                string  `json:"origin"`
	TotalRevenue          int64   `json:"total_revenue"`
	RevenuePercentage     float64 `json:"revenue_percentage"`
	PayingCustomers       uint64  `json:"paying_customers"`
	UniqueVisitors        uint64  `json:"unique_visitors"`
	ConversionRate        float64 `json:"conversion_rate"`
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"`
	PaymentCount          uint64  `json:"payment_count"`
	Currency              string  `json:"currency,omitempty"`
}

type AttributionByUTMRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	UTMType   string    `query:"utm_type"`
}

type AttributionByUTMResponse struct {
	Data []UTMAttributionDataPoint `json:"data"`
}

type UTMAttributionDataPoint struct {
	UTMValue              string  `json:"utm_value"`
	TotalRevenue          int64   `json:"total_revenue"`
	RevenuePercentage     float64 `json:"revenue_percentage"`
	PayingCustomers       uint64  `json:"paying_customers"`
	UniqueVisitors        uint64  `json:"unique_visitors"`
	ConversionRate        float64 `json:"conversion_rate"`
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"`
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
	TotalRevenue int64     `json:"total_revenue"`
	PaymentCount uint64    `json:"payment_count"`
	Currency     string    `json:"currency,omitempty"`
}

type TopCustomersRequest struct {
	ProjectID string    `query:"project_id" validate:"required"`
	TimeRange TimeRange `query:"time_range" validate:"required"`
	Limit     int       `query:"limit"`
}

type TopCustomersResponse struct {
	Customers []TopCustomer `json:"customers"`
	Total     int           `json:"total"`
}

type TopCustomer struct {
	CustomerID         string    `json:"customer_id"`
	VisitorID          string    `json:"visitor_id"`
	VisitorIDs         []string  `json:"visitor_ids,omitempty"`
	UserID             *string   `json:"user_id,omitempty"`
	ExternalID         *string   `json:"external_id,omitempty"`
	Email              *string   `json:"email,omitempty"`
	Name               *string   `json:"name,omitempty"`
	TotalRevenue       int64     `json:"total_revenue"`
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
	VisitorID  string  `json:"visitor_id"`
	UserID     *string `json:"user_id,omitempty"`
	ExternalID *string `json:"external_id,omitempty"`
	Email      *string `json:"email,omitempty"`
	Name       *string `json:"name,omitempty"`

	TotalRevenue     int64      `json:"total_revenue"`
	PaymentCount     uint64     `json:"payment_count"`
	FirstPaymentDate *time.Time `json:"first_payment_date,omitempty"`
	LastPaymentDate  *time.Time `json:"last_payment_date,omitempty"`
	AvgOrderValue    int64      `json:"avg_order_value"`
	Currency         string     `json:"currency,omitempty"`

	FirstTrafficOrigin *string `json:"first_traffic_origin,omitempty"`
	FirstUTMSource     *string `json:"first_utm_source,omitempty"`
	FirstUTMMedium     *string `json:"first_utm_medium,omitempty"`
	FirstUTMCampaign   *string `json:"first_utm_campaign,omitempty"`

	Payments []Payment `json:"payments"`

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
	ConversionRate  float64 `json:"conversion_rate"`

	AvgTimeToFirstPurchase    float64 `json:"avg_time_to_first_purchase_hours"`
	MedianTimeToFirstPurchase float64 `json:"median_time_to_first_purchase_hours"`

	CustomerLifetimeValue float64 `json:"customer_lifetime_value"`

	RepeatPurchaseRate      float64 `json:"repeat_purchase_rate"`
	AvgPurchasesPerCustomer float64 `json:"avg_purchases_per_customer"`
}

type CohortRevenueMetricsRequest struct {
	ProjectID  string    `json:"project_id" validate:"required"`
	VisitorIDs []string  `json:"visitor_ids" validate:"required,min=1"`
	TimeRange  TimeRange `json:"time_range,omitempty"`
}

type CohortRevenueMetricsResponse struct {
	TotalVisitors   uint64 `json:"total_visitors"`
	TotalCustomers  uint64 `json:"total_customers"`
	PayingCustomers uint64 `json:"paying_customers"`

	TotalRevenue          int64   `json:"total_revenue"`
	AvgRevenuePerCustomer float64 `json:"avg_revenue_per_customer"`
	AvgRevenuePerVisitor  float64 `json:"avg_revenue_per_visitor"`
	Currency              string  `json:"currency,omitempty"`

	TotalPayments          uint64  `json:"total_payments"`
	AvgPaymentsPerCustomer float64 `json:"avg_payments_per_customer"`
	AvgOrderValue          float64 `json:"avg_order_value"`
	ConversionRate         float64 `json:"conversion_rate"`
	VisitorConversionRate  float64 `json:"visitor_conversion_rate"`

	AvgTimeToFirstPurchase    float64 `json:"avg_time_to_first_purchase_hours"`
	MedianTimeToFirstPurchase float64 `json:"median_time_to_first_purchase_hours"`

	IdentifiedCustomers uint64 `json:"identified_customers"`
	AnonymousCustomers  uint64 `json:"anonymous_customers"`
}
