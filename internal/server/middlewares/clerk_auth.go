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

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/labstack/echo/v4"
)

type ClerkAuthMiddleware struct {
	secretKey           string
	OrganizationService *orgServices.OrganizationService
}

func NewClerkAuthMiddleware(
	cfg *config.Config,
	orgService *orgServices.OrganizationService,
) (*ClerkAuthMiddleware, error) {
	if cfg.ClerkSecretKey == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY is required for Clerk authentication")
	}

	// Set the Clerk API key globally
	clerk.SetKey(cfg.ClerkSecretKey)

	return &ClerkAuthMiddleware{
		secretKey:           cfg.ClerkSecretKey,
		OrganizationService: orgService,
	}, nil
}

func (m *ClerkAuthMiddleware) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing token")
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Verify the JWT token using Clerk's JWT verification
			claims, err := jwt.Verify(context.Background(), &jwt.VerifyParams{
				Token: tokenString,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Invalid token: %v", err))
			}

			// Extract user ID from claims (subject)
			userID := claims.Subject
			if userID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing user ID in token")
			}

			// Set account ID to Clerk user ID
			c.Set("account_id", userID)
			c.Set("clerk_user_id", userID)

			reqCtx := ctx.NewCtx(c)

			// Note: Email is not available in Clerk JWT claims by default
			// If you need user email, you can fetch it from Clerk API using the user ID
			// or include it in custom claims
			reqCtx.SetUser(&models.Account{
				ID:    userID,
				Email: "", // Email not available in JWT claims
			})

			// Set organization ID from Clerk claims
			// Clerk uses ActiveOrganizationID for the current organization context
			if claims.ActiveOrganizationID != "" {
				reqCtx.SetOrgID(claims.ActiveOrganizationID)
			}

			c.Set("ctx", reqCtx)

			return next(c)
		}
	}
}
