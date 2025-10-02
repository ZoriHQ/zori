package web

import (
	"marker/internal/server"
	"marker/services/auth/services"
)

// RegisterRoutes registers all auth routes with the server wrapper
func RegisterRoutes(s *server.Server, authService *services.AuthService) {
	// Auth routes group
	auth := s.Group("/api/v1/auth")

	// Register route
	auth.POST("/register", authService.Register)

	// Login route
	auth.POST("/login", authService.Login)

	// Token refresh route
	auth.POST("/refresh", authService.RefreshToken)

	// Logout route
	auth.POST("/logout", authService.Logout)

	// Password recovery routes
	auth.POST("/recover", authService.Recover)
	auth.POST("/recover-confirm", authService.RecoverConfirm)
}
