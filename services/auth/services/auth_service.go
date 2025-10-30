package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"zori/internal/config"
	"zori/internal/ctx"
	"zori/services/auth/data"
	"zori/services/auth/types"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type OSSAuthClaims struct {
	jwt.RegisteredClaims
	OrgID string `json:"org_id"`
}

type AuthService struct {
	authData *data.AuthData
	cfg      *config.Config
}

func NewAuthService(authData *data.AuthData, cfg *config.Config) *AuthService {
	return &AuthService{
		authData: authData,
		cfg:      cfg,
	}
}

// Login validates credentials and returns a JWT token
// @Summary      Login to OSS deployment
// @Description  Authenticate with admin credentials and receive a JWT token (OSS mode only)
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        credentials  body      types.LoginRequest   true  "Admin credentials"
// @Success      200          {object}  types.LoginResponse  "JWT token and organization ID"
// @Failure      400          {object}  map[string]string    "Invalid request body"
// @Failure      401          {object}  map[string]string    "Invalid credentials"
// @Failure      500          {object}  map[string]string    "Internal server error"
// @Router       /api/v1/auth/login [post]
func (s *AuthService) Login(c *ctx.Ctx) (*types.LoginResponse, error) {
	var req types.LoginRequest
	if err := c.Echo.Bind(&req); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Username and password are required")
	}

	system, err := s.authData.GetSystemConfig(context.Background())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve system configuration")
	}

	if system == nil || system.AdminUsername == nil || system.AdminPasswordHash == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "System not initialized")
	}

	if *system.AdminUsername != req.Username {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*system.AdminPasswordHash), []byte(req.Password))
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
	}

	if system.DefaultOrgID == nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Organization not configured")
	}

	token, err := s.generateToken(*system.DefaultOrgID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate token")
	}

	return &types.LoginResponse{
		Token: token,
		OrgID: *system.DefaultOrgID,
	}, nil
}

func (s *AuthService) generateToken(orgID string) (string, error) {
	claims := &OSSAuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "zori-oss",
		},
		OrgID: orgID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
