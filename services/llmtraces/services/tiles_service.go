package services

import (
	"zori/internal/ctx"
	"zori/services/analytics/filters"
	"zori/services/llmtraces/tiles"
)

// LLMTracesTilesService provides methods for managing LLM traces analytics tiles.
// It contains endpoint definitions for various tiles available on the LLM traces dashboard.
type LLMTracesTilesService struct {
	avgPricePerCustomerTile *tiles.AvgPricePerCustomerTile
	dailyPriceTile          *tiles.DailyPriceTile
	dailyPriceTimelineTile  *tiles.DailyPriceTimelineTile
}

func NewLLMTracesTilesService(
	avgPricePerCustomerTile *tiles.AvgPricePerCustomerTile,
	dailyPriceTile *tiles.DailyPriceTile,
	dailyPriceTimelineTile *tiles.DailyPriceTimelineTile,
) *LLMTracesTilesService {
	return &LLMTracesTilesService{
		avgPricePerCustomerTile: avgPricePerCustomerTile,
		dailyPriceTile:          dailyPriceTile,
		dailyPriceTimelineTile:  dailyPriceTimelineTile,
	}
}

// GetAvgPricePerCustomerTile returns the average LLM cost per unique customer
// @Summary Get average price per customer tile
// @Description Get average LLM API cost per unique customer for the current period compared to the previous period
// @Tags LLM Traces
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.AvgPricePerCustomerResponse "Average price per customer with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/llm-traces/tiles/avg-price-per-customer [get]
func (s *LLMTracesTilesService) GetAvgPricePerCustomerTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.AvgPricePerCustomerResponse, error) {
	return s.avgPricePerCustomerTile.Fetch(ctx, filter)
}

// GetDailyPriceTile returns total LLM costs and usage metrics for the period
// @Summary Get daily price tile
// @Description Get total LLM API costs, token counts, and trace count for the current period compared to the previous period
// @Tags LLM Traces
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.DailyPriceResponse "Daily price metrics with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/llm-traces/tiles/daily-price [get]
func (s *LLMTracesTilesService) GetDailyPriceTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.DailyPriceResponse, error) {
	return s.dailyPriceTile.Fetch(ctx, filter)
}

// GetDailyPriceTimelineTile returns LLM costs over time for chart visualization
// @Summary Get daily price timeline
// @Description Get LLM API costs over time broken down by day for chart visualization
// @Tags LLM Traces
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.DailyPriceTimelineResponse "Daily price timeline data for charts"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/llm-traces/tiles/timeline [get]
func (s *LLMTracesTilesService) GetDailyPriceTimelineTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.DailyPriceTimelineResponse, error) {
	return s.dailyPriceTimelineTile.Fetch(ctx, filter)
}
