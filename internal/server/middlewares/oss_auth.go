package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"zori/internal/config"
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type OSSAuthClaims struct {
	jwt.RegisteredClaims
	OrgID string `json:"org_id"`
}

type OSSAuthMiddleware struct {
	jwtSecret string
}

func NewOSSAuthMiddleware(cfg *config.Config) *OSSAuthMiddleware {
	return &OSSAuthMiddleware{
		jwtSecret: cfg.JWTSecret,
	}
}

func (m *OSSAuthMiddleware) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing token")
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse and validate the JWT
			token, err := jwt.ParseWithClaims(tokenString, &OSSAuthClaims{}, func(token *jwt.Token) (interface{}, error) {
				// Verify signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(m.jwtSecret), nil
			})

			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Invalid token: %v", err))
			}

			if !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			claims, ok := token.Claims.(*OSSAuthClaims)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token claims")
			}

			// Verify org ID exists
			if claims.OrgID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing organization ID in token")
			}

			// Set context values
			reqCtx := ctx.NewCtx(c)

			// Create a minimal account representation
			reqCtx.SetUser(&models.Account{
				ID:    "oss-admin", // OSS mode doesn't have user IDs, using a static identifier
				Email: "admin@localhost",
			})

			// Set organization ID from JWT claims
			reqCtx.SetOrgID(claims.OrgID)

			c.Set("ctx", reqCtx)

			return next(c)
		}
	}
}
