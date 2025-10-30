package web

import (
	"zori/internal/server"
	"zori/services/auth/services"
)

func RegisterRoutes(s *server.Server, authService *services.AuthService) {
	authRouteGroup := s.Group("/api/v1/auth")

	// POST /api/v1/auth/login - No authentication required
	server.GroupPOST(authRouteGroup, "/login", authService.Login)
}
