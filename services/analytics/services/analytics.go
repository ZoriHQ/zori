package services

import (
	"fmt"
	"zori/internal/ctx"
	"zori/services/analytics/data"
	"zori/services/analytics/types"

	"github.com/labstack/echo/v4"
)

type AnalyticsService struct {
	data *data.AnalyticsData
}

func NewAnalyticsService(data *data.AnalyticsData) *AnalyticsService {
	return &AnalyticsService{data: data}
}

// GetVisitorsByDevice returns visitor statistics grouped by device type over time
// @Summary Get visitors by device type
// @Description Get visitor counts grouped by device type (mobile, desktop, tablet) over a specified time range
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.VisitorsByDeviceResponse "Visitors grouped by device type"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/device [get]
func (s *AnalyticsService) GetVisitorsByDevice(c *ctx.Ctx) (*types.VisitorsByDeviceResponse, error) {
	var req types.VisitorsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	// Validate time range
	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dataPoints, err := s.data.GetVisitorsByDevice(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get visitors by device: %w", err)
	}

	return &types.VisitorsByDeviceResponse{
		Data: dataPoints,
	}, nil
}

// GetUniqueVisitorsByOrigin returns unique visitor counts grouped by traffic origin
// @Summary Get unique visitors by traffic origin
// @Description Get unique visitor counts grouped by referrer domain (traffic source)
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.VisitorsByOriginResponse "Unique visitors grouped by origin"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/origin [get]
func (s *AnalyticsService) GetUniqueVisitorsByOrigin(c *ctx.Ctx) (*types.VisitorsByOriginResponse, error) {
	var req types.VisitorsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	// Validate time range
	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dataPoints, err := s.data.GetUniqueVisitorsByOrigin(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get visitors by origin: %w", err)
	}

	return &types.VisitorsByOriginResponse{
		Data: dataPoints,
	}, nil
}

// GetUniqueVisitorsByCountry returns unique visitor counts grouped by country
// @Summary Get unique visitors by country
// @Description Get unique visitor counts grouped by country code
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.VisitorsByCountryResponse "Unique visitors grouped by country"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/country [get]
func (s *AnalyticsService) GetUniqueVisitorsByCountry(c *ctx.Ctx) (*types.VisitorsByCountryResponse, error) {
	var req types.VisitorsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	// Validate time range
	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dataPoints, err := s.data.GetUniqueVisitorsByCountry(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get visitors by country: %w", err)
	}

	return &types.VisitorsByCountryResponse{
		Data: dataPoints,
	}, nil
}

// GetRecentEvents returns the most recent events for a project
// @Summary Get recent events
// @Description Get a list of the most recent events (default: 15 events)
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param limit query int false "Maximum number of events to return (default: 15)"
// @Success 200 {object} types.RecentEventsResponse "List of recent events"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/events/recent [get]
func (s *AnalyticsService) GetRecentEvents(c *ctx.Ctx) (*types.RecentEventsResponse, error) {
	projectID := c.Echo.QueryParam("project_id")
	if projectID == "" {
		return nil, echo.NewHTTPError(400, "project_id is required")
	}

	limit := 15
	if limitParam := c.Echo.QueryParam("limit"); limitParam != "" {
		if _, err := fmt.Sscanf(limitParam, "%d", &limit); err != nil {
			limit = 15
		}
	}

	events, err := s.data.GetRecentEvents(c.Echo.Request().Context(), projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	return &types.RecentEventsResponse{
		Events: events,
		Total:  len(events),
	}, nil
}

// GetTopVisitors returns the most active visitors for a project
// @Summary Get top visitors
// @Description Get a list of the most active visitors ranked by event count
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Param limit query int false "Maximum number of visitors to return (default: 50)"
// @Success 200 {object} types.TopVisitorsResponse "List of top visitors"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/top [get]
func (s *AnalyticsService) GetTopVisitors(c *ctx.Ctx) (*types.TopVisitorsResponse, error) {
	var req types.TopVisitorsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	// Validate time range
	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	// Default limit
	if req.Limit <= 0 {
		req.Limit = 50
	}

	visitors, err := s.data.GetTopVisitors(c.Echo.Request().Context(), req.ProjectID, req.TimeRange, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top visitors: %w", err)
	}

	return &types.TopVisitorsResponse{
		Visitors: visitors,
		Total:    len(visitors),
	}, nil
}

// GetVisitorProfile returns detailed profile for a single visitor
// @Summary Get visitor profile
// @Description Get detailed information about a specific visitor including their event history and aggregated statistics
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param visitor_id query string true "Visitor ID"
// @Success 200 {object} types.VisitorProfileResponse "Visitor profile details"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 404 {object} map[string]interface{} "Visitor not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/profile [get]
func (s *AnalyticsService) GetVisitorProfile(c *ctx.Ctx) (*types.VisitorProfileResponse, error) {
	projectID := c.Echo.QueryParam("project_id")
	if projectID == "" {
		return nil, echo.NewHTTPError(400, "project_id is required")
	}

	visitorID := c.Echo.QueryParam("visitor_id")
	if visitorID == "" {
		return nil, echo.NewHTTPError(400, "visitor_id is required")
	}

	profile, err := s.data.GetVisitorProfile(c.Echo.Request().Context(), projectID, visitorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get visitor profile: %w", err)
	}

	return profile, nil
}

// GetUniqueVisitorsTimeline returns unique visitors over time split by device type
// @Summary Get unique visitors timeline
// @Description Get unique visitor counts over time, split by mobile and desktop devices for chart visualization
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.UniqueVisitorsTimelineResponse "Unique visitors timeline data"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/timeline [get]
func (s *AnalyticsService) GetUniqueVisitorsTimeline(c *ctx.Ctx) (*types.UniqueVisitorsTimelineResponse, error) {
	var req types.VisitorsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	// Validate time range
	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dataPoints, err := s.data.GetUniqueVisitorsTimeline(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique visitors timeline: %w", err)
	}

	return &types.UniqueVisitorsTimelineResponse{
		Data: dataPoints,
	}, nil
}
