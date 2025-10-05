package web

import (
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/services/organizations/services"
)

func RegisterRoutes(
	accountService *services.AccountService,
	organizationService *services.OrganizationService,
	s *server.Server,
	jwtMiddleware *middlewares.JwtMiddleware,
) {
	g := s.Group("/api/v1/organization")
	g.Use(jwtMiddleware.Middleware())

	server.GroupGET(g, "/", organizationService.GetOrganization)
}
