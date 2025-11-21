package services

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/services/analytics/data"
	"zori/services/analytics/filters"
	"zori/services/analytics/types"
	ingestionData "zori/services/ingestion/data"
	projectsData "zori/services/projects/data"

	"github.com/labstack/echo/v4"
)

type VisitorsService struct {
	visitorRepository *ingestionData.VisitorRepository
	projectData       *projectsData.ProjectData
	data              *data.AnalyticsData
}

func NewVisitorsService(
	visitorRepository *ingestionData.VisitorRepository,
	projectData *projectsData.ProjectData,
	data *data.AnalyticsData,
) *VisitorsService {
	return &VisitorsService{
		visitorRepository: visitorRepository,
		projectData:       projectData,
		data:              data,
	}
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
func (s *VisitorsService) IdentifyVisitor(c *ctx.Ctx) (*types.ManualIdentifyResponse, error) {
	var req types.ManualIdentifyRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(400, "Invalid request parameters")
	}

	if req.ProjectID == "" {
		return nil, echo.NewHTTPError(400, "project_id is required")
	}
	if req.VisitorID == "" {
		return nil, echo.NewHTTPError(400, "visitor_id is required")
	}

	if req.UserID == nil && req.ExternalID == nil && req.Email == nil && req.Name == nil && req.Phone == nil && len(req.AdditionalProperties) == 0 {
		return nil, echo.NewHTTPError(400, "At least one identity field must be provided")
	}

	visitor := &models.Visitor{
		VisitorID:    req.VisitorID,
		ProjectID:    req.ProjectID,
		UserID:       req.UserID,
		ExternalID:   req.ExternalID,
		Email:        req.Email,
		Name:         req.Name,
		Phone:        req.Phone,
		CustomTraits: make(map[string]any),
	}

	if len(req.AdditionalProperties) > 0 {
		visitor.CustomTraits = req.AdditionalProperties
	}

	project, err := s.projectData.GetProjectByID(c.Echo.Request().Context(), req.ProjectID)
	if err != nil {
		return nil, echo.NewHTTPError(400, fmt.Sprintf("Failed to find project: %v", err))
	}
	visitor.OrganizationID = project.OrganizationID

	if err := s.visitorRepository.UpsertVisitor(c.Echo.Request().Context(), visitor); err != nil {
		return nil, fmt.Errorf("failed to identify visitor: %w", err)
	}

	return &types.ManualIdentifyResponse{
		Success:   true,
		Message:   "Visitor identified successfully",
		VisitorID: req.VisitorID,
	}, nil
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
