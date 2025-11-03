package services

import (
	"fmt"
	"zori/internal/ctx"
	"zori/services/revenue/data"
	"zori/services/revenue/types"

	"github.com/labstack/echo/v4"
)

type RevenueService struct {
	data *data.RevenueData
}

func NewRevenueService(data *data.RevenueData) *RevenueService {
	return &RevenueService{
		data: data,
	}
}

// GetDashboard returns key revenue metrics for the dashboard
// @Summary Get revenue dashboard
// @Description Get key revenue metrics including total revenue, conversion rate, and average order value
// @Tags Revenue
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.DashboardResponse "Revenue dashboard metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/revenue/dashboard [get]
func (s *RevenueService) GetDashboard(c *ctx.Ctx) (*types.DashboardResponse, error) {
	var req types.DashboardRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dashboard, err := s.data.GetDashboardMetrics(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue dashboard: %w", err)
	}

	return dashboard, nil
}

// GetAttributionByOrigin returns revenue attributed to traffic origins
// @Summary Get revenue attribution by traffic origin
// @Description Get revenue metrics grouped by traffic origin (referrer domain) using first-touch attribution
// @Tags Revenue
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.AttributionByOriginResponse "Revenue attribution by origin"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/revenue/attribution/origin [get]
func (s *RevenueService) GetAttributionByOrigin(c *ctx.Ctx) (*types.AttributionByOriginResponse, error) {
	var req types.AttributionByOriginRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dataPoints, err := s.data.GetAttributionByOrigin(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get attribution by origin: %w", err)
	}

	return &types.AttributionByOriginResponse{
		Data: dataPoints,
	}, nil
}

// GetAttributionByUTM returns revenue attributed to UTM parameters
// @Summary Get revenue attribution by UTM parameters
// @Description Get revenue metrics grouped by UTM source, medium, or campaign using first-touch attribution
// @Tags Revenue
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Param utm_type query string false "UTM type: source, medium, or campaign (default: source)"
// @Success 200 {object} types.AttributionByUTMResponse "Revenue attribution by UTM parameters"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/revenue/attribution/utm [get]
func (s *RevenueService) GetAttributionByUTM(c *ctx.Ctx) (*types.AttributionByUTMResponse, error) {
	var req types.AttributionByUTMRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	if req.UTMType == "" {
		req.UTMType = "source"
	}

	dataPoints, err := s.data.GetAttributionByUTM(c.Echo.Request().Context(), req.ProjectID, req.TimeRange, req.UTMType)
	if err != nil {
		return nil, fmt.Errorf("failed to get attribution by UTM: %w", err)
	}

	return &types.AttributionByUTMResponse{
		Data: dataPoints,
	}, nil
}

// GetTimeline returns revenue over time
// @Summary Get revenue timeline
// @Description Get revenue metrics over time for chart visualization
// @Tags Revenue
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.TimelineResponse "Revenue timeline data"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/revenue/timeline [get]
func (s *RevenueService) GetTimeline(c *ctx.Ctx) (*types.TimelineResponse, error) {
	var req types.TimelineRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dataPoints, err := s.data.GetTimeline(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue timeline: %w", err)
	}

	return &types.TimelineResponse{
		Data: dataPoints,
	}, nil
}

// GetTopCustomers returns the highest revenue customers
// @Summary Get top revenue customers
// @Description Get a list of customers ranked by total revenue
// @Tags Revenue
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Param limit query int false "Maximum number of customers to return (default: 50)"
// @Success 200 {object} types.TopCustomersResponse "List of top customers"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/revenue/customers/top [get]
func (s *RevenueService) GetTopCustomers(c *ctx.Ctx) (*types.TopCustomersResponse, error) {
	var req types.TopCustomersRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}

	customers, err := s.data.GetTopCustomers(c.Echo.Request().Context(), req.ProjectID, req.TimeRange, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top customers: %w", err)
	}

	return &types.TopCustomersResponse{
		Customers: customers,
		Total:     len(customers),
	}, nil
}

// GetCustomerProfile returns detailed revenue profile for a specific customer
// @Summary Get customer revenue profile
// @Description Get detailed revenue information for a specific customer including payment history and attribution
// @Tags Revenue
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param visitor_id query string true "Visitor ID"
// @Success 200 {object} types.CustomerProfileResponse "Customer revenue profile"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 404 {object} map[string]interface{} "Customer not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/revenue/customers/profile [get]
func (s *RevenueService) GetCustomerProfile(c *ctx.Ctx) (*types.CustomerProfileResponse, error) {
	projectID := c.Echo.QueryParam("project_id")
	if projectID == "" {
		return nil, echo.NewHTTPError(400, "project_id is required")
	}

	visitorID := c.Echo.QueryParam("visitor_id")
	if visitorID == "" {
		return nil, echo.NewHTTPError(400, "visitor_id is required")
	}

	profile, err := s.data.GetCustomerProfile(c.Echo.Request().Context(), projectID, visitorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer profile: %w", err)
	}

	return profile, nil
}

// GetConversionMetrics returns conversion funnel and customer value metrics
// @Summary Get conversion metrics
// @Description Get conversion funnel metrics including conversion rate, time to purchase, and customer lifetime value
// @Tags Revenue
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.ConversionMetricsResponse "Conversion metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/revenue/conversion/metrics [get]
func (s *RevenueService) GetConversionMetrics(c *ctx.Ctx) (*types.ConversionMetricsResponse, error) {
	var req types.ConversionMetricsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast30Days
	}

	metrics, err := s.data.GetConversionMetrics(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversion metrics: %w", err)
	}

	return metrics, nil
}

// GetCohortRevenueMetrics returns revenue metrics for a specific cohort of visitors
// @Summary Get cohort revenue metrics
// @Description Analyze revenue metrics for a specific cohort of visitors, including total revenue, conversion rates, and time to first purchase
// @Tags Revenue
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body types.CohortRevenueMetricsRequest true "Cohort analysis request with visitor IDs"
// @Success 200 {object} types.CohortRevenueMetricsResponse "Cohort revenue metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/revenue/cohort/metrics [post]
func (s *RevenueService) GetCohortRevenueMetrics(c *ctx.Ctx) (*types.CohortRevenueMetricsResponse, error) {
	var req types.CohortRevenueMetricsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.ProjectID == "" {
		return nil, echo.NewHTTPError(400, "project_id is required")
	}

	if len(req.VisitorIDs) == 0 {
		return nil, echo.NewHTTPError(400, "visitor_ids is required and must not be empty")
	}

	metrics, err := s.data.GetCohortRevenueMetrics(c.Echo.Request().Context(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get cohort revenue metrics: %w", err)
	}

	return metrics, nil
}
