package web

import (
	"marker/internal/server"
	"marker/internal/server/middlewares"
	"marker/services/projects/services"
)

func RegisterRoutes(s *server.Server, projectService *services.ProjectService, jwtMiddleware *middlewares.JwtMiddleware) {
	projectRouteGroup := s.Group("/projects")
	projectRouteGroup.Use(jwtMiddleware.Middleware())
	server.GroupGET(projectRouteGroup, "/list", projectService.ListProjects)
}
