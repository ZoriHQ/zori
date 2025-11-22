package services

import (
	"fmt"
	"time"

	"zori/internal/ctx"
	"zori/services/analytics/filters"
	"zori/services/llmtraces/data"
	"zori/services/llmtraces/types"
)

type LLMTracesService struct {
	repository *data.LLMTraceRepository
}

func NewLLMTracesService(repository *data.LLMTraceRepository) *LLMTracesService {
	return &LLMTracesService{
		repository: repository,
	}
}

// GetTraces returns a paginated list of LLM traces
// @Summary Get LLM traces list
// @Description Get a paginated list of LLM traces with filtering by customer, visitor, provider, and model
// @Tags LLM Traces
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Param time_boundaries query string false "Time range (last_day, last_week, last_month, last_3_months, last_year)"
// @Param customer_id query string false "Filter by customer ID"
// @Param visitor_id query string false "Filter by visitor ID"
// @Param provider query string false "Filter by provider (e.g., openai, anthropic)"
// @Param model query string false "Filter by model name"
// @Param limit query int false "Number of traces to return (default 50, max 100)"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} types.LLMTracesListResponse "List of traces with pagination info"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/llm-traces/list [get]
func (s *LLMTracesService) GetTraces(ctx *ctx.Ctx, req *types.LLMTracesListRequest) (*types.LLMTracesListResponse, error) {
	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Validate time boundaries
	timeBoundaries := filters.TimeBoundaries(req.TimeBoundaries)
	if timeBoundaries == "" {
		timeBoundaries = filters.TimeBoundariesLastWeek
	}

	timeRange, err := filters.ValidateTimeRange(timeBoundaries)
	if err != nil {
		return nil, fmt.Errorf("invalid time boundaries: %w", err)
	}

	traces, totalCount, err := s.repository.GetTraces(
		ctx,
		ctx.OrgID(),
		req.ProjectID,
		timeRange.Start,
		time.Now().UTC(),
		req,
	)
	if err != nil {
		return nil, err
	}

	return &types.LLMTracesListResponse{
		Traces:     traces,
		TotalCount: totalCount,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}, nil
}

// GetFilterOptions returns available filter options for traces
// @Summary Get LLM trace filter options
// @Description Get available filter options (providers, models, customer IDs) for the trace list
// @Tags LLM Traces
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters including project_id and time range"
// @Success 200 {object} types.LLMTraceFilterOptions "Available filter options"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/llm-traces/filter-options [get]
func (s *LLMTracesService) GetFilterOptions(ctx *ctx.Ctx, filter *filters.SectionFilter) (*types.LLMTraceFilterOptions, error) {
	return s.repository.GetFilterOptions(
		ctx,
		ctx.OrgID(),
		filter.ProjectID,
		filter.TimeRange.Start,
		time.Now().UTC(),
	)
}
