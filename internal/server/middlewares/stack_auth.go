package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"zori/internal/config"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	orgServices "zori/services/organizations/services"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type StackAuthClaims struct {
	jwt.RegisteredClaims
	// Stack Auth custom claims
	Email          string `json:"email,omitempty"`
	SelectedTeamID string `json:"selected_team_id,omitempty"`
	EmailVerified  bool   `json:"email_verified,omitempty"`
}

type StackAuthMiddleware struct {
	jwks                keyfunc.Keyfunc
	OrganizationService *orgServices.OrganizationService
	cancelFunc          context.CancelFunc
}

func NewStackAuthMiddleware(
	cfg *config.Config,
	orgService *orgServices.OrganizationService,
) (*StackAuthMiddleware, error) {
	jwksURL := cfg.GetStackAuthJWKSURL()

	// Create a context that can cancel the JWKS background refresh
	ctx, cancel := context.WithCancel(context.Background())

	// Create the JWKS from the remote URL with automatic refresh
	// NewDefaultCtx creates a keyfunc with default options and automatic refresh
	jwks, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create JWKS: %w", err)
	}

	return &StackAuthMiddleware{
		jwks:                jwks,
		OrganizationService: orgService,
		cancelFunc:          cancel,
	}, nil
}

// Cleanup stops the background JWKS refresh goroutine
func (m *StackAuthMiddleware) Cleanup() {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
}

func (m *StackAuthMiddleware) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing token")
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse and validate the JWT
			token, err := jwt.ParseWithClaims(tokenString, &StackAuthClaims{}, m.jwks.Keyfunc)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Invalid token: %v", err))
			}

			if !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			claims, ok := token.Claims.(*StackAuthClaims)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token claims")
			}

			// The subject (sub) claim contains the Stack Auth user ID
			stackAuthUserID := claims.Subject
			if stackAuthUserID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing user ID in token")
			}

			// For now, we'll use the Stack Auth user ID as the account ID
			// You may want to create a mapping table later
			c.Set("account_id", stackAuthUserID)
			c.Set("stack_auth_user_id", stackAuthUserID)

			// Extract email if present
			if claims.Email != "" {
				c.Set("email", claims.Email)
			}

			reqCtx := ctx.NewCtx(c)

			reqCtx.SetUser(&models.Account{
				ID:    stackAuthUserID,
				Email: claims.Email,
			})

			// Set organization ID from Stack Auth (external provider)
			if claims.SelectedTeamID != "" {
				reqCtx.SetOrgID(claims.SelectedTeamID)
			}

			c.Set("ctx", reqCtx)

			return next(c)
		}
	}
}
