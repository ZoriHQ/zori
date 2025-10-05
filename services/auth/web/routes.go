package web

import (
	"marker/internal/server"
	"marker/services/auth/services"
)

func RegisterRoutes(s *server.Server, authService *services.AuthService) {
	auth := s.Group("/api/v1/auth")

	auth.POST("/register", authService.Register)
	auth.POST("/login", authService.Login)
	auth.POST("/refresh", authService.RefreshToken)
	auth.POST("/logout", authService.Logout)
	auth.POST("/recover", authService.Recover)
	auth.POST("/recover-confirm", authService.RecoverConfirm)
}
