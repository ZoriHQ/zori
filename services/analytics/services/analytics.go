package services

import (
	"fmt"
	"zori/internal/ctx"
	"zori/services/analytics/data"
	"zori/services/analytics/filters"
	"zori/services/analytics/types"
	ingestionData "zori/services/ingestion/data"
	projectsData "zori/services/projects/data"
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
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.VisitorsByDeviceResponse "Visitors grouped by device type"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/device [get]
func (s *AnalyticsService) GetVisitorsByDevice(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.VisitorsByDeviceResponse, error) {
	dataPoints, err := s.data.GetVisitorsByDevice(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get visitors by device: %w", err)
	}

	return &types.VisitorsByDeviceResponse{
		Data: dataPoints,
	}, nil
}

// GetRecentEvents returns the most recent events for a project with optional filters
// @Summary Get recent events
// @Description Get a list of recent events with optional filters (visitor_id, user_id, external_id, traffic_origin, page_path, event_name)
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query types.RecentEventsRequest true "Filter parameters"
// @Success 200 {object} types.RecentEventsResponse "List of recent events with pagination info"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/events/recent [get]
func (s *AnalyticsService) GetRecentEvents(ctx *ctx.Ctx, filter *types.RecentEventsRequest) (*types.RecentEventsResponse, error) {
	events, totalCount, err := s.data.GetRecentEvents(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	if len(events) == 0 {
		events = make([]types.RecentEvent, 0)
	}

	return &types.RecentEventsResponse{
		Events: events,
		Total:  totalCount,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

// GetEventFilterOptions returns available filter options for events
// @Summary Get event filter options
// @Description Get unique traffic origins, page paths, and event names to populate filter dropdowns
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.EventFilterOptionsResponse "Filter options for traffic origins, pages, and event names"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/events/filter-options [get]
func (s *AnalyticsService) GetEventFilterOptions(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.EventFilterOptionsResponse, error) {
	filterOptions, err := s.data.GetEventFilterOptions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get event filter options: %w", err)
	}

	return filterOptions, nil
}

// GetTopVisitors returns the most active visitors grouped by identified information with payment metrics
// @Summary Get top visitors with payment data
// @Description Get top visitors grouped by identified information (user_id, external_id, email) with payment metrics including revenue, distinct payments, and time to first purchase
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.TopVisitorsResponse "List of top visitors with payment data"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/top [get]
func (s *AnalyticsService) GetTopVisitors(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.TopVisitorsResponse, error) {
	visitors, err := s.data.GetTopVisitors(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get top visitors: %w", err)
	}

	return &types.TopVisitorsResponse{
		Visitors: visitors,
		Total:    len(visitors),
	}, nil
}

// GetSessionMetrics returns session duration and pages per session metrics
// @Summary Get session metrics
// @Description Get session metrics including average duration and pages per session
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.SessionMetricsResponse "Session metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/sessions/metrics [get]
func (s *AnalyticsService) GetSessionMetrics(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.SessionMetricsResponse, error) {
	metrics, err := s.data.GetSessionMetrics(ctx, filter)
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
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.BounceRateResponse "Bounce rate metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/sessions/bounce-rate [get]
func (s *AnalyticsService) GetBounceRate(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.BounceRateResponse, error) {
	bounceRate, err := s.data.GetBounceRate(ctx, filter)
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
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.ActiveUsersResponse "Active user metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/users/active [get]
func (s *AnalyticsService) GetActiveUsers(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.ActiveUsersResponse, error) {
	activeUsers, err := s.data.GetActiveUsers(ctx, filter)
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
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.ReturnRateResponse "Return rate metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/retention/return-rate [get]
func (s *AnalyticsService) GetReturnRate(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.ReturnRateResponse, error) {
	returnRate, err := s.data.GetReturnRate(ctx, filter)
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
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.ChurnRateResponse "Churn rate metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/retention/churn-rate [get]
func (s *AnalyticsService) GetChurnRate(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.ChurnRateResponse, error) {
	// TODO:: replace number of days with user input
	churnRate, err := s.data.GetChurnRate(ctx, filter, 30)
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
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.CohortAnalysisResponse "Cohort analysis data"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/retention/cohorts [get]
func (s *AnalyticsService) GetCohortAnalysis(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.CohortAnalysisResponse, error) {
	cohorts, err := s.data.GetCohortAnalysis(ctx, filter)
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
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.DashboardMetricsResponse "Dashboard metrics"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/dashboard [get]
func (s *AnalyticsService) GetDashboardMetrics(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.DashboardMetricsResponse, error) {
	dashboard, err := s.data.GetDashboardMetrics(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard metrics: %w", err)
	}

	return dashboard, nil
}
