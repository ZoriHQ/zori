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

			claims, err := jwt.Verify(context.Background(), &jwt.VerifyParams{
				Token: tokenString,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Invalid token: %v", err))
			}

			userID := claims.Subject
			if userID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing user ID in token")
			}

			c.Set("account_id", userID)
			c.Set("clerk_user_id", userID)

			reqCtx := ctx.NewCtx(c)

			reqCtx.SetUser(&models.Account{
				ID:    userID,
				Email: "",
			})

			if claims.ActiveOrganizationID != "" {
				reqCtx.SetOrgID(claims.ActiveOrganizationID)
			}

			c.Set("ctx", reqCtx)

			return next(c)
		}
	}
}
