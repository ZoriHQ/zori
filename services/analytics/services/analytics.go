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

// GetLLMTraces returns a paginated list of LLM traces with optional filters
// @Summary Get LLM traces list
// @Description Get a paginated list of LLM traces with optional filters (name, user_id, session_id, model)
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query types.LLMTracesListRequest true "Filter parameters"
// @Success 200 {object} types.LLMTracesListResponse "List of LLM traces with pagination info"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/llm/traces [get]
func (s *AnalyticsService) GetLLMTraces(ctx *ctx.Ctx, filter *types.LLMTracesListRequest) (*types.LLMTracesListResponse, error) {
	traces, totalCount, err := s.data.GetLLMTraces(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM traces: %w", err)
	}

	if len(traces) == 0 {
		traces = make([]types.LLMTraceItem, 0)
	}

	return &types.LLMTracesListResponse{
		Traces: traces,
		Total:  totalCount,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

// GetLLMTraceFilterOptions returns available filter options for LLM traces
// @Summary Get LLM trace filter options
// @Description Get unique names, user_ids, session_ids, and models to populate filter dropdowns
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} types.LLMTraceFilterOptionsResponse "Filter options for LLM traces"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/llm/traces/filter-options [get]
func (s *AnalyticsService) GetLLMTraceFilterOptions(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.LLMTraceFilterOptionsResponse, error) {
	filterOptions, err := s.data.GetLLMTraceFilterOptions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM trace filter options: %w", err)
	}

	return filterOptions, nil
}
