package services

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"zori/internal/ctx"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"
	"zori/internal/utils"
	"zori/services/auth/helpers"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/uptrace/bun"
)

type RegisterRequest struct {
	Email            string `json:"email" validate:"required,email" example:"user@example.com"`
	Password         string `json:"password" validate:"required,min=8" example:"SecurePassword123!"`
	FirstName        string `json:"first_name" example:"John"`
	LastName         string `json:"last_name" example:"Doe"`
	OrganizationName string `json:"organization_name" validate:"required" example:"Acme Corporation"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"SecurePassword123!"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type RecoverRequest struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

type RecoverConfirmRequest struct {
	Token    string `json:"token" validate:"required" example:"recovery-token-from-email"`
	Password string `json:"password" validate:"required,min=8" example:"NewSecurePassword123!"`
}

type AuthResponse struct {
	AccessToken  string               `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string               `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn    int64                `json:"expires_in" example:"900"`
	Account      *models.Account      `json:"account"`
	Organization *models.Organization `json:"organization"`
}

type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

type AuthService struct {
	db       *bun.DB
	password *PasswordService
	jwt      *JWTService
	token    *TokenService
}

func NewAuthService(db *postgres.PostgresDB, password *PasswordService, jwt *JWTService, token *TokenService) *AuthService {
	return &AuthService{
		db:       db.DB,
		password: password,
		jwt:      jwt,
		token:    token,
	}
}

// Register creates a new user account and organization
// @Summary Register a new account
// @Description Create a new user account with an organization
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 200 {object} AuthResponse "Successfully registered and authenticated"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation failed"
// @Failure 409 {object} map[string]interface{} "Account with email already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/register [post]
func (s *AuthService) Register(ctx *ctx.Ctx) (*AuthResponse, error) {
	var req RegisterRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	// Check if account already exists
	exists, err := s.db.NewSelect().
		Model((*models.Account)(nil)).
		Where("email = ?", strings.ToLower(req.Email)).
		Exists(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to check if account exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("account with email %s already exists", req.Email)
	}

	hashedPassword, err := s.password.ValidateAndHashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	slug := helpers.GenerateSlug(req.OrganizationName)

	tx, err := s.db.BeginTx(ctx.Echo.Request().Context(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	account := &models.Account{
		ID:           uuid.New().String(),
		Email:        strings.ToLower(req.Email),
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = tx.NewInsert().Model(account).Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	org := &models.Organization{
		ID:        uuid.New().String(),
		Name:      req.OrganizationName,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = tx.NewInsert().Model(org).Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	member := &models.OrganizationMember{
		ID:             uuid.New().String(),
		OrganizationID: org.ID,
		AccountID:      account.ID,
		Role:           models.RoleOwner,
		JoinedAt:       time.Now(),
	}

	_, err = tx.NewInsert().Model(member).Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to create organization member: %w", err)
	}

	// Create session
	sessionID := uuid.New().String()
	session := &models.Session{
		ID:        sessionID,
		AccountID: account.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = tx.NewInsert().Model(session).Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate JWT tokens with session ID
	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(
		sessionID,
		account.ID,
		org.ID,
		account.Email,
		models.RoleOwner,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * 60),
		Account:      account,
		Organization: org,
	}, nil
}

// Login authenticates a user with email and password
// @Summary User login
// @Description Authenticate a user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse "Successfully authenticated"
// @Failure 400 {object} map[string]interface{} "Invalid email or password"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/login [post]
func (s *AuthService) Login(ctx *ctx.Ctx) (*AuthResponse, error) {
	var req LoginRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	account := &models.Account{}
	err := s.db.NewSelect().
		Model(account).
		Where("email = ?", strings.ToLower(req.Email)).
		Scan(ctx.Echo.Request().Context())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("Invalid email or password"))
	}

	err = s.password.VerifyPassword(account.PasswordHash, req.Password)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("Invalid email or password"))
	}

	member := &models.OrganizationMember{}
	err = s.db.NewSelect().
		Model(member).
		Relation("Organization").
		Where("om.account_id = ?", account.ID).
		Where("om.role IN (?)", bun.In([]string{models.RoleOwner, models.RoleAdmin})).
		Limit(1).
		Scan(ctx.Echo.Request().Context())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("Organization not found"))
	}

	// Create new session
	sessionID := uuid.New().String()
	session := &models.Session{
		ID:        sessionID,
		AccountID: account.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = s.db.NewInsert().Model(session).Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(
		sessionID,
		account.ID,
		member.OrganizationID,
		account.Email,
		member.Role,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * 60), // 15 minutes
		Account:      account,
		Organization: member.Organization,
	}, nil
}

// RefreshToken exchanges a valid refresh token for new access and refresh tokens
// @Summary Refresh access token
// @Description Exchange a valid refresh token for new access and refresh tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} AuthResponse "Successfully refreshed tokens"
// @Failure 400 {object} map[string]interface{} "Invalid or expired refresh token"
// @Failure 401 {object} map[string]interface{} "Session not found or expired"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/refresh [post]
func (s *AuthService) RefreshToken(ctx *ctx.Ctx) (*AuthResponse, error) {
	var req RefreshRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	// Validate refresh token JWT
	refreshClaims, err := s.jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if session exists and is valid
	session := &models.Session{}
	err = s.db.NewSelect().
		Model(session).
		Relation("Account").
		Where("s.id = ?", refreshClaims.SessionID).
		Where("s.account_id = ?", refreshClaims.AccountID).
		Scan(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("session not found or expired")
	}

	if session.IsExpired() {
		// Delete expired session
		s.db.NewDelete().Model(session).WherePK().Exec(ctx.Echo.Request().Context())
		return nil, fmt.Errorf("session expired")
	}

	// Get organization membership
	member := &models.OrganizationMember{}
	err = s.db.NewSelect().
		Model(member).
		Relation("Organization").
		Where("om.account_id = ?", session.AccountID).
		Where("om.role IN (?)", bun.In([]string{models.RoleOwner, models.RoleAdmin})).
		Limit(1).
		Scan(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("no organization found for user")
	}

	// Update session expiry
	session.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	session.UpdatedAt = time.Now()
	_, err = s.db.NewUpdate().
		Model(session).
		Column("expires_at", "updated_at").
		WherePK().
		Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Generate new token pair
	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(
		session.ID,
		session.Account.ID,
		member.OrganizationID,
		session.Account.Email,
		member.Role,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * 60), // 15 minutes
		Account:      session.Account,
		Organization: member.Organization,
	}, nil
}

// Logout invalidates the current session and refresh token
// @Summary User logout
// @Description Invalidate the current session and refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "Logout request"
// @Success 200 {object} MessageResponse "Successfully logged out"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Router /api/v1/auth/logout [post]
func (s *AuthService) Logout(ctx *ctx.Ctx) (*MessageResponse, error) {
	var req LogoutRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	// Validate refresh token to get session ID
	refreshClaims, err := s.jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		// Still try to return success even if token is invalid
		return &MessageResponse{Message: "Logged out successfully"}, nil
	}

	// Delete session by ID
	_, err = s.db.NewDelete().
		Model((*models.Session)(nil)).
		Where("id = ?", refreshClaims.SessionID).
		Where("account_id = ?", refreshClaims.AccountID).
		Exec(ctx.Echo.Request().Context())
	if err != nil {
		// Log error but still return success
		// The session might already be deleted or expired
	}

	return &MessageResponse{Message: "Logged out successfully"}, nil
}

// Recover initiates password recovery by sending an email to the user
// @Summary Request password recovery
// @Description Send a password recovery email to the registered email address
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RecoverRequest true "Recovery request"
// @Success 200 {object} MessageResponse "Recovery email sent if account exists"
// @Failure 400 {object} map[string]interface{} "Invalid email format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/recover [post]
func (s *AuthService) Recover(ctx *ctx.Ctx) (*MessageResponse, error) {
	var req RecoverRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	// TODO: Implement password recovery
	return &MessageResponse{Message: "Password recovery not implemented yet"}, nil
}

// RecoverConfirm resets the password using a recovery token
// @Summary Confirm password recovery
// @Description Reset password using recovery token received via email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RecoverConfirmRequest true "Recovery confirmation"
// @Success 200 {object} MessageResponse "Password successfully reset"
// @Failure 400 {object} map[string]interface{} "Invalid or expired token"
// @Failure 422 {object} map[string]interface{} "Password validation failed"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/auth/recover-confirm [post]
func (s *AuthService) RecoverConfirm(ctx *ctx.Ctx) (*MessageResponse, error) {
	var req RecoverConfirmRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	// TODO: Implement password recovery confirmation
	return &MessageResponse{Message: "Password recovery confirmation not implemented yet"}, nil
}
