package services

import (
	"context"
	"fmt"
	"net/http"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/internal/utils"
	"zori/services/projects/data"
	"zori/services/projects/types"

	"github.com/labstack/echo/v4"
)

type ListProjectsResponse struct {
	Projects []*models.Project `json:"projects"`
	Total    int               `json:"total" example:"10"`
}

type ProjectResponse struct {
	*models.Project
}

type CreateProjectResponse struct {
	*models.Project
}

type ProjectService struct {
	data *data.ProjectData
}

func NewProjectService(data *data.ProjectData) *ProjectService {
	return &ProjectService{data: data}
}

func (p *ProjectService) GetProjectByPublishableToken(token string) (*models.Project, error) {
	return p.data.GetProjectByPublishableToken(token)
}

func (p *ProjectService) GetProjectByLangfusePublicKey(ctx context.Context, publicKey string) (*models.Project, error) {
	return p.data.GetProjectByLangfusePublicKey(ctx, publicKey)
}

func (p *ProjectService) SetFirstEventReceivedNow(projectID string) error {
	err := p.data.SetFirstEventReceived(projectID)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// @Summary List organization projects
// @Description Get a list of all projects belonging to the authenticated user's organization
// @Tags Projects
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} services.ListProjectsResponse "List of projects"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/projects/list [get]
func (s *ProjectService) ListProjects(c *ctx.Ctx) (*ListProjectsResponse, error) {
	projects, err := s.data.ListOrganizationProjects(c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return &ListProjectsResponse{
		Projects: projects,
		Total:    len(projects),
	}, nil
}

// @Summary Create a new project
// @Description Create a new project for the authenticated user's organization. The response includes Langfuse API keys for LLM trace ingestion.
// @Tags Projects
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body types.CreateProjectRequest true "Project creation details"
// @Success 201 {object} services.CreateProjectResponse "Created project with Langfuse credentials"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 409 {object} map[string]interface{} "Project with this name already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/projects [post]
func (s *ProjectService) CreateProject(c *ctx.Ctx) (*CreateProjectResponse, error) {
	var req types.CreateProjectRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := s.data.CreateProject(c, &req)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	c.Echo.Response().Status = http.StatusCreated

	return &CreateProjectResponse{
		Project: result.Project,
	}, nil
}

// @Summary Update a project
// @Description Update an existing project's details
// @Tags Projects
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Project ID"
// @Param request body types.UpdateProjectRequest true "Project update details"
// @Success 200 {object} services.ProjectResponse "Updated project"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 404 {object} map[string]interface{} "Project not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/projects/{id} [put]
func (s *ProjectService) UpdateProject(c *ctx.Ctx) (*ProjectResponse, error) {
	projectID := c.Echo.Param("id")
	if projectID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	var req types.UpdateProjectRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	exists, err := s.data.ProjectExists(c.Echo.Request().Context(), projectID, c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to check project existence: %w", err)
	}
	if !exists {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	project, err := s.data.UpdateProject(c.Echo.Request().Context(), projectID, c.OrgID(), &req)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return &ProjectResponse{Project: project}, nil
}

// @Summary Get a project
// @Description Get a single project by its ID
// @Tags Projects
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Project ID"
// @Success 200 {object} services.ProjectResponse "Project details"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 404 {object} map[string]interface{} "Project not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/projects/{id} [get]
func (s *ProjectService) GetProject(c *ctx.Ctx) (*ProjectResponse, error) {
	projectID := c.Echo.Param("id")
	if projectID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	return s.GetProjectNoContext(c, projectID, c.OrgID())
}

func (s *ProjectService) GetProjectNoContext(ctx context.Context, projectID, orgID string) (*ProjectResponse, error) {
	project, err := s.data.GetProject(ctx, projectID, orgID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, echo.NewHTTPError(http.StatusNotFound, "Project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &ProjectResponse{Project: project}, nil
}

// @Summary Delete a project
// @Description Delete a project and all its associated data
// @Tags Projects
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Project ID"
// @Success 200 {object} map[string]string "Deletion confirmation"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 404 {object} map[string]interface{} "Project not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/projects/{id} [delete]
func (s *ProjectService) DeleteProject(c *ctx.Ctx) (map[string]string, error) {
	projectID := c.Echo.Param("id")
	if projectID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	rowsAffected, err := s.data.DeleteProject(c, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete project: %w", err)
	}

	if rowsAffected == 0 {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	return map[string]string{
		"message": "Project deleted successfully",
	}, nil
}
