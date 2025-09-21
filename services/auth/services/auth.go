package services

import (
	"fmt"
	"strings"
	"time"

	"marker/internal/server"
	"marker/internal/utils"
	"marker/services/auth/models"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuthService struct {
	db       *bun.DB
	password *PasswordService
	jwt      *JWTService
	token    *TokenService
}

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

type AuthResponse struct {
	AccessToken  string               `json:"access_token"`
	RefreshToken string               `json:"refresh_token"`
	ExpiresIn    int64                `json:"expires_in"`
	Account      *models.Account      `json:"account"`
	Organization *models.Organization `json:"organization"`
}

func NewAuthService(db *bun.DB, password *PasswordService, jwt *JWTService, token *TokenService) *AuthService {
	return &AuthService{
		db:       db,
		password: password,
		jwt:      jwt,
		token:    token,
	}
}

func (s *AuthService) Register(ctx *server.Ctx) (any, error) {
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

	slug := generateSlug(req.OrganizationName)

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

	// Generate JWT tokens
	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(
		account.ID,
		org.ID,
		account.Email,
		models.RoleOwner,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	session := &models.Session{
		ID:           uuid.New().String(),
		AccountID:    account.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt:    time.Now(),
	}

	_, err = tx.NewInsert().Model(session).Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
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

func (s *AuthService) Login(ctx *server.Ctx) (any, error) {
	var req LoginRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}
	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	// Find account by email
	account := &models.Account{}
	err := s.db.NewSelect().
		Model(account).
		Where("email = ?", strings.ToLower(req.Email)).
		Scan(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password
	err = s.password.VerifyPassword(account.PasswordHash, req.Password)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	member := &models.OrganizationMember{}
	err = s.db.NewSelect().
		Model(member).
		Relation("Organization").
		Where("om.account_id = ?", account.ID).
		Where("om.role IN (?)", bun.In([]string{models.RoleOwner, models.RoleAdmin})).
		Order("om.role ASC, om.joined_at ASC").
		Limit(1).
		Scan(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("no organization found for user")
	}

	// Generate new JWT tokens
	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(
		account.ID,
		member.OrganizationID,
		account.Email,
		member.Role,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	session := &models.Session{
		ID:           uuid.New().String(),
		AccountID:    account.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:    time.Now(),
	}

	_, err = s.db.NewInsert().Model(session).Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * 60), // 15 minutes
		Account:      account,
		Organization: member.Organization,
	}, nil
}

func (s *AuthService) RefreshToken(ctx *server.Ctx) (any, error) {
	type RefreshRequest struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	var req RefreshRequest
	if err := ctx.Echo.Bind(&req); err != nil {
		return nil, err
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	refreshToken := req.RefreshToken
	if !s.token.IsValidRefreshToken(refreshToken) {
		return nil, fmt.Errorf("invalid refresh token format")
	}

	session := &models.Session{}
	err := s.db.NewSelect().
		Model(session).
		Relation("Account").
		Where("s.refresh_token = ?", refreshToken).
		Scan(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if session.IsExpired() {
		s.db.NewDelete().Model(session).WherePK().Exec(ctx.Echo.Request().Context())
		return nil, fmt.Errorf("refresh token expired")
	}

	member := &models.OrganizationMember{}
	err = s.db.NewSelect().
		Model(member).
		Relation("Organization").
		Where("om.account_id = ?", session.AccountID).
		Where("om.role IN (?)", bun.In([]string{models.RoleOwner, models.RoleAdmin})).
		Order("om.role ASC, om.joined_at ASC").
		Limit(1).
		Scan(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("no organization found for user")
	}

	// Generate new access token
	accessToken, err := s.jwt.GenerateAccessToken(
		session.Account.ID,
		member.OrganizationID,
		session.Account.Email,
		member.Role,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,   // Return same refresh token
		ExpiresIn:    int64(15 * 60), // 15 minutes
		Account:      session.Account,
		Organization: member.Organization,
	}, nil
}

func (s *AuthService) Logout(ctx *server.Ctx) (any, error) {
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
	// Delete session by refresh token
	_, err := s.db.NewDelete().
		Model((*models.Session)(nil)).
		Where("refresh_token = ?", req.RefreshToken).
		Exec(ctx.Echo.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to logout: %w", err)
	}

	return map[string]string{"message": "Logged out successfully"}, nil
}

func (s *AuthService) Recover(ctx *server.Ctx) (any, error) {
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

func (s *AuthService) RecoverConfirm(ctx *server.Ctx) (any, error) {
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

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Add timestamp to ensure uniqueness
	return fmt.Sprintf("%s-%d", slug, time.Now().Unix())
}
