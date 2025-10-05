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

	server.GroupPUT(projectRouteGroup, "/:id", projectService.UpdateProject)

	server.GroupDELETE(projectRouteGroup, "/:id", projectService.DeleteProject)
}
