package middlewares

import (
	"marker/services/auth/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

func JwtMiddleware(jwtService *services.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("Authorization")
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing token")
			}

			claims, err := jwtService.ValidateAccessToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			c.Set("account_id", claims.AccountID)
			c.Set("organization_id", claims.OrganizationID)
			return next(c)
		}
	}
}
