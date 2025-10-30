package web

import (
	"zori/internal/server"
	"zori/services/auth/services"
)

func RegisterRoutes(s *server.Server, authService *services.AuthService) {
	authRouteGroup := s.Group("/api/v1/auth")

	server.GroupPOST(authRouteGroup, "/login", authService.Login)
}
