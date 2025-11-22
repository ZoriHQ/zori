package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/internal/utils"
	"zori/services/analytics/filters"
	"zori/services/funnels/data"
	"zori/services/funnels/types"
	projectsData "zori/services/projects/data"

	"github.com/labstack/echo/v4"
)

type FunnelService struct {
	repository    *data.FunnelRepository
	analyticsData *data.FunnelAnalyticsData
	projectData   *projectsData.ProjectData
}

func NewFunnelService(
	repository *data.FunnelRepository,
	analyticsData *data.FunnelAnalyticsData,
	projectData *projectsData.ProjectData,
) *FunnelService {
	return &FunnelService{
		repository:    repository,
		analyticsData: analyticsData,
		projectData:   projectData,
	}
}

// @Summary Create a new funnel
// @Description Create a new funnel configuration for tracking conversion paths
// @Tags Funnels
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body types.CreateFunnelRequest true "Funnel configuration"
// @Success 201 {object} types.FunnelResponse "Created funnel"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 409 {object} map[string]interface{} "Funnel with this name already exists"
// @Router /api/v1/funnels [post]
func (s *FunnelService) CreateFunnel(c *ctx.Ctx) (*types.FunnelResponse, error) {
	var req types.CreateFunnelRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	project, err := s.projectData.GetProject(c.Echo.Request().Context(), req.ProjectID, c.OrgID())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	existing, _ := s.repository.GetFunnelByProjectAndName(c.Echo.Request().Context(), req.ProjectID, req.Name)
	if existing != nil && existing.ID != "" {
		return nil, echo.NewHTTPError(http.StatusConflict, "Funnel with this name already exists")
	}

	windowSeconds := 86400
	if req.WindowSeconds != nil {
		windowSeconds = *req.WindowSeconds
	}

	isStrict := false
	if req.IsStrict != nil {
		isStrict = *req.IsStrict
	}

	funnel := &models.Funnel{
		ProjectID:      project.ID,
		OrganizationID: project.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		FunnelType:     req.FunnelType,
		WindowSeconds:  windowSeconds,
		IsStrict:       isStrict,
	}

	if req.FunnelType == models.FunnelTypeDepth && req.DepthConfig != nil {
		depthConfigJSON, err := json.Marshal(req.DepthConfig)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid depth config")
		}
		funnel.DepthConfig = depthConfigJSON
	}

	if err := s.repository.CreateFunnel(c.Echo.Request().Context(), funnel); err != nil {
		return nil, fmt.Errorf("failed to create funnel: %w", err)
	}

	if req.FunnelType == models.FunnelTypeSequential && len(req.Steps) > 0 {
		steps := make([]*models.FunnelStep, len(req.Steps))
		for i, stepReq := range req.Steps {
			conditionJSON, err := json.Marshal(stepReq.Condition)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid step condition")
			}
			steps[i] = &models.FunnelStep{
				FunnelID:  funnel.ID,
				StepOrder: i + 1,
				Name:      stepReq.Name,
				Condition: conditionJSON,
			}
		}

		if err := s.repository.CreateFunnelSteps(c.Echo.Request().Context(), steps); err != nil {
			return nil, fmt.Errorf("failed to create funnel steps: %w", err)
		}

		funnel.Steps = steps
	}

	c.Echo.Response().Status = http.StatusCreated
	return types.ToFunnelResponse(funnel), nil
}

// @Summary List funnels
// @Description Get all funnels for a project
// @Tags Funnels
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param project_id query string true "Project ID"
// @Success 200 {object} types.ListFunnelsResponse "List of funnels"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /api/v1/funnels [get]
func (s *FunnelService) ListFunnels(c *ctx.Ctx, req *types.ListFunnelsRequest) (*types.ListFunnelsResponse, error) {
	funnels, err := s.repository.ListFunnelsByProject(c.Echo.Request().Context(), req.ProjectID, c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to list funnels: %w", err)
	}

	responses := make([]*types.FunnelResponse, len(funnels))
	for i, funnel := range funnels {
		responses[i] = types.ToFunnelResponse(funnel)
	}

	return &types.ListFunnelsResponse{
		Funnels: responses,
		Total:   len(responses),
	}, nil
}

// @Summary Get a funnel
// @Description Get a single funnel by ID
// @Tags Funnels
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Funnel ID"
// @Success 200 {object} types.FunnelResponse "Funnel details"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Funnel not found"
// @Router /api/v1/funnels/{id} [get]
func (s *FunnelService) GetFunnel(c *ctx.Ctx) (*types.FunnelResponse, error) {
	funnelID := c.Echo.Param("id")
	if funnelID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Funnel ID is required")
	}

	funnel, err := s.repository.GetFunnel(c.Echo.Request().Context(), funnelID, c.OrgID())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Funnel not found")
	}

	return types.ToFunnelResponse(funnel), nil
}

// @Summary Update a funnel
// @Description Update funnel configuration
// @Tags Funnels
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Funnel ID"
// @Param request body types.UpdateFunnelRequest true "Update details"
// @Success 200 {object} types.FunnelResponse "Updated funnel"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Funnel not found"
// @Router /api/v1/funnels/{id} [put]
func (s *FunnelService) UpdateFunnel(c *ctx.Ctx) (*types.FunnelResponse, error) {
	funnelID := c.Echo.Param("id")
	if funnelID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Funnel ID is required")
	}

	var req types.UpdateFunnelRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	exists, err := s.repository.FunnelExists(c.Echo.Request().Context(), funnelID, c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to check funnel existence: %w", err)
	}
	if !exists {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Funnel not found")
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.FunnelType != nil {
		updates["funnel_type"] = *req.FunnelType
	}
	if req.WindowSeconds != nil {
		updates["window_seconds"] = *req.WindowSeconds
	}
	if req.IsStrict != nil {
		updates["is_strict"] = *req.IsStrict
	}
	if req.DepthConfig != nil {
		depthConfigJSON, err := json.Marshal(req.DepthConfig)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid depth config")
		}
		updates["depth_config"] = depthConfigJSON
	}

	if len(updates) > 0 {
		_, err = s.repository.UpdateFunnel(c.Echo.Request().Context(), funnelID, c.OrgID(), updates)
		if err != nil {
			return nil, fmt.Errorf("failed to update funnel: %w", err)
		}
	}

	if req.Steps != nil {
		if err := s.repository.DeleteFunnelSteps(c.Echo.Request().Context(), funnelID); err != nil {
			return nil, fmt.Errorf("failed to delete existing steps: %w", err)
		}

		if len(req.Steps) > 0 {
			steps := make([]*models.FunnelStep, len(req.Steps))
			for i, stepReq := range req.Steps {
				conditionJSON, err := json.Marshal(stepReq.Condition)
				if err != nil {
					return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid step condition")
				}
				steps[i] = &models.FunnelStep{
					FunnelID:  funnelID,
					StepOrder: i + 1,
					Name:      stepReq.Name,
					Condition: conditionJSON,
				}
			}

			if err := s.repository.CreateFunnelSteps(c.Echo.Request().Context(), steps); err != nil {
				return nil, fmt.Errorf("failed to create funnel steps: %w", err)
			}
		}
	}

	funnel, err := s.repository.GetFunnel(c.Echo.Request().Context(), funnelID, c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to get updated funnel: %w", err)
	}

	return types.ToFunnelResponse(funnel), nil
}

// @Summary Delete a funnel
// @Description Delete a funnel configuration
// @Tags Funnels
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Funnel ID"
// @Success 200 {object} map[string]string "Deletion confirmation"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Funnel not found"
// @Router /api/v1/funnels/{id} [delete]
func (s *FunnelService) DeleteFunnel(c *ctx.Ctx) (map[string]string, error) {
	funnelID := c.Echo.Param("id")
	if funnelID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Funnel ID is required")
	}

	exists, err := s.repository.FunnelExists(c.Echo.Request().Context(), funnelID, c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to check funnel existence: %w", err)
	}
	if !exists {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Funnel not found")
	}

	if err := s.repository.DeleteFunnel(c.Echo.Request().Context(), funnelID, c.OrgID()); err != nil {
		return nil, fmt.Errorf("failed to delete funnel: %w", err)
	}

	return map[string]string{
		"message": "Funnel deleted successfully",
	}, nil
}

// @Summary Analyze a funnel
// @Description Run funnel analysis and get conversion metrics
// @Tags Funnels
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Funnel ID"
// @Param filter query types.AnalyzeFunnelRequest true "Analysis parameters"
// @Success 200 {object} types.UnifiedFunnelAnalysisResponse "Funnel analysis results"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Funnel not found"
// @Router /api/v1/funnels/{id}/analyze [get]
func (s *FunnelService) AnalyzeFunnel(c *ctx.Ctx, req *types.AnalyzeFunnelRequest) (*types.UnifiedFunnelAnalysisResponse, error) {
	funnelID := c.Echo.Param("id")
	if funnelID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Funnel ID is required")
	}

	funnel, err := s.repository.GetFunnel(c.Echo.Request().Context(), funnelID, c.OrgID())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Funnel not found")
	}

	if funnel.ProjectID != req.ProjectID {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Project ID mismatch")
	}

	timeRange, err := filters.ValidateTimeRange(filters.TimeBoundaries(req.TimeRange))
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	switch funnel.FunnelType {
	case models.FunnelTypeSequential:
		result, err := s.analyticsData.AnalyzeSequentialFunnel(c, funnel, timeRange.Start)
		if err != nil {
			return nil, err
		}
		return &types.UnifiedFunnelAnalysisResponse{
			FunnelID:          result.FunnelID,
			FunnelName:        result.FunnelName,
			FunnelType:        models.FunnelTypeSequential,
			AnalyzedAt:        result.AnalyzedAt,
			TotalVisitors:     result.TotalVisitors,
			ConvertedVisitors: result.ConvertedVisitors,
			OverallConversion: result.OverallConversion,
			Steps:             result.Steps,
		}, nil
	case models.FunnelTypeDepth:
		result, err := s.analyticsData.AnalyzeDepthFunnel(c, funnel, timeRange.Start)
		if err != nil {
			return nil, err
		}
		return &types.UnifiedFunnelAnalysisResponse{
			FunnelID:          result.FunnelID,
			FunnelName:        result.FunnelName,
			FunnelType:        models.FunnelTypeDepth,
			AnalyzedAt:        result.AnalyzedAt,
			TotalSessions:     result.TotalSessions,
			DepthDistribution: result.DepthDistribution,
		}, nil
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Unknown funnel type")
	}
}
