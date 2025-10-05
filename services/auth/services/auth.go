package services

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"marker/internal/ctx"
	"marker/internal/storage/postgres"
	"marker/internal/storage/postgres/models"
	"marker/internal/utils"
	"marker/services/auth/helpers"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/uptrace/bun"
)

type RegisterRequest struct {
	Email            string `json:"email" validate:"required,email"`
	Password         string `json:"password" validate:"required,min=8"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	OrganizationName string `json:"organization_name" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string               `json:"access_token"`
	RefreshToken string               `json:"refresh_token"`
	ExpiresIn    int64                `json:"expires_in"`
	Account      *models.Account      `json:"account"`
	Organization *models.Organization `json:"organization"`
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

func (s *AuthService) Register(ctx *ctx.Ctx) (any, error) {
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

func (s *AuthService) Login(ctx *ctx.Ctx) (any, error) {
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

func (s *AuthService) RefreshToken(ctx *ctx.Ctx) (any, error) {
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

func (s *AuthService) Logout(ctx *ctx.Ctx) (any, error) {
	type LogoutRequest struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

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
		return map[string]string{"message": "Logged out successfully"}, nil
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

	return map[string]string{"message": "Logged out successfully"}, nil
}

func (s *AuthService) Recover(ctx *ctx.Ctx) (any, error) {
	type RecoverRequest struct {
		Email string `json:"email" validate:"required,email"`
	}

	var req RecoverRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	// TODO: Implement password recovery
	return map[string]string{"message": "Password recovery not implemented yet"}, nil
}

func (s *AuthService) RecoverConfirm(ctx *ctx.Ctx) (any, error) {
	type RecoverConfirmRequest struct {
		Token    string `json:"token" validate:"required"`
		Password string `json:"password" validate:"required,min=8"`
	}

	var req RecoverConfirmRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	// TODO: Implement password recovery confirmation
	return map[string]string{"message": "Password recovery confirmation not implemented yet"}, nil
}
