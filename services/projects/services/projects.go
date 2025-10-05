package services

import (
	"fmt"
	"net/http"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/internal/utils"
	"zori/services/projects/data"
	"zori/services/projects/types"

	"github.com/labstack/echo/v4"
)

// ListProjectsResponse represents the response for listing projects
type ListProjectsResponse struct {
	Projects []*models.Project `json:"projects"`
	Total    int               `json:"total" example:"10"`
}

// ProjectResponse represents a single project response
type ProjectResponse struct {
	*models.Project
}

type ProjectService struct {
	data *data.ProjectData
}

func NewProjectService(data *data.ProjectData) *ProjectService {
	return &ProjectService{data: data}
}

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

func (s *ProjectService) CreateProject(c *ctx.Ctx) (*ProjectResponse, error) {
	var req types.CreateProjectRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	project, err := s.data.CreateProject(c, &req)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return &ProjectResponse{Project: project}, nil
}

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

	// Check if project exists
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

func (s *ProjectService) GetProject(c *ctx.Ctx) (*ProjectResponse, error) {
	projectID := c.Echo.Param("id")
	if projectID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	project, err := s.data.GetProject(c.Echo.Request().Context(), projectID, c.OrgID())
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, echo.NewHTTPError(http.StatusNotFound, "Project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &ProjectResponse{Project: project}, nil
}

func (s *ProjectService) DeleteProject(c *ctx.Ctx) (map[string]string, error) {
	projectID := c.Echo.Param("id")
	if projectID == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	// Check if project exists
	exists, err := s.data.ProjectExists(c.Echo.Request().Context(), projectID, c.OrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to check project existence: %w", err)
	}
	if !exists {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	err = s.data.DeleteProject(c, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete project: %w", err)
	}

	return map[string]string{
		"message": "Project deleted successfully",
	}, nil
}
