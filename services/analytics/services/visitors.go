package services

import (
	"fmt"
	"zori/internal/ctx"
	"zori/services/analytics/data"
	"zori/services/analytics/filters"
	"zori/services/analytics/types"
)

type VisitorsService struct {
	data *data.AnalyticsData
}

func NewVisitorsService(
	data *data.AnalyticsData,
) *VisitorsService {
	return &VisitorsService{
		data: data,
	}
}

// GetVisitorProfile returns detailed profile for a single visitor
// @Summary Get visitor profile
// @Description Get detailed information about a specific visitor including their event history and aggregated statistics
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.VisitorProfileFilter true "Filter parameters"
// @Success 200 {object} types.VisitorProfileResponse "Visitor profile details"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 404 {object} map[string]interface{} "Visitor not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/profile [get]
func (s *VisitorsService) GetVisitorProfile(ctx *ctx.Ctx, filter *filters.VisitorProfileFilter) (*types.VisitorProfileResponse, error) {
	profile, err := s.data.GetVisitorProfile(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get visitor profile: %w", err)
	}

	return profile, nil
}
