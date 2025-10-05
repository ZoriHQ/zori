package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/projects/services"
)

func RegisterRoutes(s *server.Server, projectService *services.ProjectService, jwtMiddleware *middlewares.JwtMiddleware) {
	projectRouteGroup := s.Group("/api/v1/projects")
	projectRouteGroup.Use(jwtMiddleware.Middleware())

	server.GroupGET(projectRouteGroup, "/list", projectService.ListProjects)

	server.GroupGET(projectRouteGroup, "/:id", projectService.GetProject)

	server.GroupPOST(projectRouteGroup, "", projectService.CreateProject)

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
	server.GroupPUT(projectRouteGroup, "/:id", projectService.UpdateProject)

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
	server.GroupDELETE(projectRouteGroup, "/:id", projectService.DeleteProject)
}
