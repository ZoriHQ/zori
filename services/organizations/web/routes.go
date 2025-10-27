package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/organizations/services"
)

func RegisterRoutes(
	organizationService *services.OrganizationService,
	s *server.Server,
	stackAuthMiddleware *middlewares.StackAuthMiddleware,
) {
	g := s.Group("/api/v1/organization")
	g.Use(stackAuthMiddleware.Middleware())

	server.GroupGET(g, "/", organizationService.GetOrganization)
}
