package web

import (
	"zori/internal/server"
	"zori/services/auth/services"
)

func RegisterRoutes(s *server.Server, authService *services.AuthService) {
	auth := s.Group("/api/v1/auth")

	// @Summary Register a new account
	// @Description Create a new user account with an organization
	// @Tags Authentication
	// @Accept json
	// @Produce json
	// @Param request body services.RegisterRequest true "Registration details"
	// @Success 200 {object} services.AuthResponse "Successfully registered and authenticated"
	// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
	// @Failure 409 {object} map[string]interface{} "Account with email already exists"
	// @Failure 500 {object} map[string]interface{} "Internal server error"
	// @Router /auth/register [post]
	server.GroupPOST(auth, "/register", authService.Register)

	// @Summary User login
	// @Description Authenticate a user with email and password
	// @Tags Authentication
	// @Accept json
	// @Produce json
	// @Param request body services.LoginRequest true "Login credentials"
	// @Success 200 {object} services.AuthResponse "Successfully authenticated"
	// @Failure 400 {object} map[string]interface{} "Invalid email or password"
	// @Failure 500 {object} map[string]interface{} "Internal server error"
	// @Router /auth/login [post]
	server.GroupPOST(auth, "/login", authService.Login)

	// @Summary Refresh access token
	// @Description Exchange a valid refresh token for new access and refresh tokens
	// @Tags Authentication
	// @Accept json
	// @Produce json
	// @Param request body services.RefreshRequest true "Refresh token"
	// @Success 200 {object} services.AuthResponse "Successfully refreshed tokens"
	// @Failure 400 {object} map[string]interface{} "Invalid or expired refresh token"
	// @Failure 401 {object} map[string]interface{} "Session not found or expired"
	// @Failure 500 {object} map[string]interface{} "Internal server error"
	// @Router /auth/refresh [post]
	server.GroupPOST(auth, "/refresh", authService.RefreshToken)

	// @Summary User logout
	// @Description Invalidate the current session and refresh token
	// @Tags Authentication
	// @Accept json
	// @Produce json
	// @Param request body services.LogoutRequest true "Logout request"
	// @Success 200 {object} services.MessageResponse "Successfully logged out"
	// @Failure 400 {object} map[string]interface{} "Invalid request"
	// @Router /auth/logout [post]
	server.GroupPOST(auth, "/logout", authService.Logout)

	// @Summary Request password recovery
	// @Description Send a password recovery email to the registered email address
	// @Tags Authentication
	// @Accept json
	// @Produce json
	// @Param request body services.RecoverRequest true "Recovery request"
	// @Success 200 {object} services.MessageResponse "Recovery email sent if account exists"
	// @Failure 400 {object} map[string]interface{} "Invalid email format"
	// @Failure 500 {object} map[string]interface{} "Internal server error"
	// @Router /auth/recover [post]
	server.GroupPOST(auth, "/recover", authService.Recover)

	// @Summary Confirm password recovery
	// @Description Reset password using recovery token received via email
	// @Tags Authentication
	// @Accept json
	// @Produce json
	// @Param request body services.RecoverConfirmRequest true "Recovery confirmation"
	// @Success 200 {object} services.MessageResponse "Password successfully reset"
	// @Failure 400 {object} map[string]interface{} "Invalid or expired token"
	// @Failure 422 {object} map[string]interface{} "Password validation failed"
	// @Failure 500 {object} map[string]interface{} "Internal server error"
	// @Router /auth/recover-confirm [post]
	server.GroupPOST(auth, "/recover-confirm", authService.RecoverConfirm)
}
