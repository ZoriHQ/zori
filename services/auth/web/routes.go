package web

import (
	"zori/internal/server"
	"zori/services/auth/services"
)

func RegisterRoutes(s *server.Server, authService *services.AuthService) {
	auth := s.Group("/api/v1/auth")

	server.GroupPOST(auth, "/register", authService.Register)

	server.GroupPOST(auth, "/login", authService.Login)

	server.GroupPOST(auth, "/refresh", authService.RefreshToken)

	server.GroupPOST(auth, "/logout", authService.Logout)

	server.GroupPOST(auth, "/recover", authService.Recover)

	server.GroupPOST(auth, "/recover-confirm", authService.RecoverConfirm)
}
