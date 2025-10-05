package web

import (
	"marker/internal/server"
	"marker/internal/server/middlewares"
	"marker/services/organizations/services"
)

func RegisterRoutes(
	accountService *services.AccountService,
	organizationService *services.OrganizationService,
	s *server.Server,
	jwtMiddleware *middlewares.JwtMiddleware,
) {
	g := s.Group("/api/v1/organization")
	g.Use(jwtMiddleware.Middleware())

	g.GET("/", organizationService.GetOrganization)
}
