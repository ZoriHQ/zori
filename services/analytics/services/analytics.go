package services

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/services/analytics/data"
	"zori/services/analytics/types"
	ingestionData "zori/services/ingestion/data"
	projectsData "zori/services/projects/data"

	"github.com/labstack/echo/v4"
)

type AnalyticsService struct {
	data              *data.AnalyticsData
	visitorRepository *ingestionData.VisitorRepository
	projectData       *projectsData.ProjectData
}

func NewAnalyticsService(data *data.AnalyticsData, visitorRepository *ingestionData.VisitorRepository, projectData *projectsData.ProjectData) *AnalyticsService {
	return &AnalyticsService{
		data:              data,
		visitorRepository: visitorRepository,
		projectData:       projectData,
	}
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

// GetRecentEvents returns the most recent events for a project with optional filters
// @Summary Get recent events
// @Description Get a list of recent events with optional filters (visitor_id, user_id, external_id, traffic_origin, page_path)
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param limit query int false "Maximum number of events to return (default: 15)"
// @Param offset query int false "Offset for pagination (default: 0)"
// @Param visitor_id query string false "Filter by visitor ID"
// @Param user_id query string false "Filter by user ID"
// @Param external_id query string false "Filter by external ID"
// @Param traffic_origin query string false "Filter by traffic origin (referrer domain)"
// @Param page_path query string false "Filter by page path"
// @Success 200 {object} types.RecentEventsResponse "List of recent events with pagination info"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/events/recent [get]
func (s *AnalyticsService) GetRecentEvents(c *ctx.Ctx) (*types.RecentEventsResponse, error) {
	var req types.RecentEventsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.ProjectID == "" {
		return nil, echo.NewHTTPError(400, "project_id is required")
	}

	events, totalCount, err := s.data.GetRecentEvents(c.Echo.Request().Context(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	return &types.RecentEventsResponse{
		Events: events,
		Total:  totalCount,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}

// GetEventFilterOptions returns available filter options for events
// @Summary Get event filter options
// @Description Get unique traffic origins and page paths to populate filter dropdowns
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string false "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.EventFilterOptionsResponse "Filter options for traffic origins and pages"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/events/filter-options [get]
func (s *AnalyticsService) GetEventFilterOptions(c *ctx.Ctx) (*types.EventFilterOptionsResponse, error) {
	var req types.EventFilterOptionsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.ProjectID == "" {
		return nil, echo.NewHTTPError(400, "project_id is required")
	}

	var timeRange *types.TimeRange
	if req.TimeRange != "" {
		timeRange = &req.TimeRange
	}

	filterOptions, err := s.data.GetEventFilterOptions(c.Echo.Request().Context(), req.ProjectID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get event filter options: %w", err)
	}

	return filterOptions, nil
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

// GetSessionMetrics returns session duration and pages per session metrics
// @Summary Get session metrics
// @Description Get session metrics including average duration and pages per session
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.SessionMetricsResponse "Session metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/sessions/metrics [get]
func (s *AnalyticsService) GetSessionMetrics(c *ctx.Ctx) (*types.SessionMetricsResponse, error) {
	var req types.SessionMetricsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	metrics, err := s.data.GetSessionMetrics(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get session metrics: %w", err)
	}

	return metrics, nil
}

// GetBounceRate returns bounce rate metrics overall and by page
// @Summary Get bounce rate
// @Description Get bounce rate metrics including overall bounce rate and per-page breakdown
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Param limit query int false "Limit for per-page breakdown (default: 20)"
// @Success 200 {object} types.BounceRateResponse "Bounce rate metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/sessions/bounce-rate [get]
func (s *AnalyticsService) GetBounceRate(c *ctx.Ctx) (*types.BounceRateResponse, error) {
	var req types.BounceRateRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	bounceRate, err := s.data.GetBounceRate(c.Echo.Request().Context(), req.ProjectID, req.TimeRange, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get bounce rate: %w", err)
	}

	return bounceRate, nil
}

// GetActiveUsers returns DAU/WAU/MAU metrics
// @Summary Get active users
// @Description Get daily, weekly, and monthly active user counts
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Success 200 {object} types.ActiveUsersResponse "Active user metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/users/active [get]
func (s *AnalyticsService) GetActiveUsers(c *ctx.Ctx) (*types.ActiveUsersResponse, error) {
	var req types.ActiveUsersRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	activeUsers, err := s.data.GetActiveUsers(c.Echo.Request().Context(), req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	return activeUsers, nil
}

// GetReturnRate returns user return rate metrics
// @Summary Get return rate
// @Description Get metrics about user return rate and time between sessions
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.ReturnRateResponse "Return rate metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/retention/return-rate [get]
func (s *AnalyticsService) GetReturnRate(c *ctx.Ctx) (*types.ReturnRateResponse, error) {
	var req types.ReturnRateRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast30Days
	}

	returnRate, err := s.data.GetReturnRate(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get return rate: %w", err)
	}

	return returnRate, nil
}

// GetChurnRate returns user churn rate metrics
// @Summary Get churn rate
// @Description Get metrics about user churn based on inactivity threshold
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Param churn_threshold_days query int false "Days of inactivity to consider churned (default: 30)"
// @Success 200 {object} types.ChurnRateResponse "Churn rate metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/retention/churn-rate [get]
func (s *AnalyticsService) GetChurnRate(c *ctx.Ctx) (*types.ChurnRateResponse, error) {
	var req types.ChurnRateRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast90Days
	}

	churnRate, err := s.data.GetChurnRate(c.Echo.Request().Context(), req.ProjectID, req.TimeRange, req.ChurnThresholdDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get churn rate: %w", err)
	}

	return churnRate, nil
}

// GetCohortAnalysis returns cohort retention analysis
// @Summary Get cohort analysis
// @Description Get cohort retention analysis showing how user groups retain over time
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.CohortAnalysisResponse "Cohort analysis data"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/retention/cohorts [get]
func (s *AnalyticsService) GetCohortAnalysis(c *ctx.Ctx) (*types.CohortAnalysisResponse, error) {
	var req types.CohortAnalysisRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast90Days
	}

	cohorts, err := s.data.GetCohortAnalysis(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get cohort analysis: %w", err)
	}

	return cohorts, nil
}

// GetDashboardMetrics returns combined key metrics for a dashboard
// @Summary Get dashboard metrics
// @Description Get combined key metrics including sessions, active users, bounce rate, and retention for dashboard display
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.DashboardMetricsResponse "Dashboard metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/dashboard [get]
func (s *AnalyticsService) GetDashboardMetrics(c *ctx.Ctx) (*types.DashboardMetricsResponse, error) {
	var req types.DashboardMetricsRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dashboard, err := s.data.GetDashboardMetrics(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard metrics: %w", err)
	}

	return dashboard, nil
}

// IdentifyVisitor manually identifies a visitor from the dashboard
// @Summary Manually identify a visitor
// @Description Manually identify a visitor by updating their profile information from the dashboard
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body types.ManualIdentifyRequest true "Identification details"
// @Success 200 {object} types.ManualIdentifyResponse "Identification successful"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/identify [post]
func (s *AnalyticsService) IdentifyVisitor(c *ctx.Ctx) (*types.ManualIdentifyResponse, error) {
	var req types.ManualIdentifyRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	// Validate required fields
	if req.ProjectID == "" {
		return nil, echo.NewHTTPError(400, "project_id is required")
	}
	if req.VisitorID == "" {
		return nil, echo.NewHTTPError(400, "visitor_id is required")
	}

	// At least one identity field should be provided
	if req.UserID == nil && req.ExternalID == nil && req.Email == nil && req.Name == nil && req.Phone == nil && (req.AdditionalProperties == nil || len(req.AdditionalProperties) == 0) {
		return nil, echo.NewHTTPError(400, "At least one identity field must be provided")
	}

	// Create visitor model for upsert
	visitor := &models.Visitor{
		VisitorID:    req.VisitorID,
		ProjectID:    req.ProjectID,
		UserID:       req.UserID,
		ExternalID:   req.ExternalID,
		Email:        req.Email,
		Name:         req.Name,
		Phone:        req.Phone,
		CustomTraits: make(map[string]interface{}),
	}

	// Add additional properties to custom traits
	if req.AdditionalProperties != nil && len(req.AdditionalProperties) > 0 {
		visitor.CustomTraits = req.AdditionalProperties
	}

	// Get organization_id from the project
	project, err := s.projectData.GetProjectByID(c.Echo.Request().Context(), req.ProjectID)
	if err != nil {
		return nil, echo.NewHTTPError(400, fmt.Sprintf("Failed to find project: %v", err))
	}
	visitor.OrganizationID = project.OrganizationID

	// Upsert visitor identity
	if err := s.visitorRepository.UpsertVisitor(c.Echo.Request().Context(), visitor); err != nil {
		return nil, fmt.Errorf("failed to identify visitor: %w", err)
	}

	return &types.ManualIdentifyResponse{
		Success:   true,
		Message:   "Visitor identified successfully",
		VisitorID: req.VisitorID,
	}, nil
}

// GetRevenueByUTM returns revenue grouped by UTM parameters
// @Summary Get revenue by UTM parameters
// @Description Get revenue metrics grouped by UTM source, medium, or campaign
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Param utm_type query string false "UTM type: source, medium, or campaign (default: source)"
// @Success 200 {object} types.RevenueByUTMResponse "Revenue by UTM parameters"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/revenue/by-utm [get]
func (s *AnalyticsService) GetRevenueByUTM(c *ctx.Ctx) (*types.RevenueByUTMResponse, error) {
	var req types.RevenueByUTMRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	// Validate time range
	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	// Default to source if not specified
	if req.UTMType == "" {
		req.UTMType = "source"
	}

	dataPoints, err := s.data.GetRevenueByUTMSource(c.Echo.Request().Context(), req.ProjectID, req.TimeRange, req.UTMType)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue by UTM: %w", err)
	}

	return &types.RevenueByUTMResponse{
		Data: dataPoints,
	}, nil
}

// GetRevenueTimeline returns revenue over time
// @Summary Get revenue timeline
// @Description Get revenue metrics over time for chart visualization
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_range query string true "Time range" Enums(last_hour, today, last_7_days, last_30_days, last_90_days)
// @Success 200 {object} types.RevenueTimelineResponse "Revenue timeline data"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/revenue/timeline [get]
func (s *AnalyticsService) GetRevenueTimeline(c *ctx.Ctx) (*types.RevenueTimelineResponse, error) {
	var req types.RevenueTimelineRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	// Validate time range
	if req.TimeRange == "" {
		req.TimeRange = types.TimeRangeLast7Days
	}

	dataPoints, err := s.data.GetRevenueTimeline(c.Echo.Request().Context(), req.ProjectID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue timeline: %w", err)
	}

	return &types.RevenueTimelineResponse{
		Data: dataPoints,
	}, nil
}
