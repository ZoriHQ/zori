package middlewares

import (
	"net/http"
	"zori/internal/ctx"
	"zori/services/auth/services"
	orgServices "zori/services/organizations/services"

	"github.com/labstack/echo/v4"
)

type JwtMiddleware struct {
	JwtService          *services.JWTService
	OrganizationService *orgServices.OrganizationService
	AccountService      *orgServices.AccountService
}

func NewJwtMiddleware(jwtService *services.JWTService,
	orgService *orgServices.OrganizationService,
	accountService *orgServices.AccountService,
) *JwtMiddleware {
	return &JwtMiddleware{
		JwtService:          jwtService,
		OrganizationService: orgService,
		AccountService:      accountService,
	}
}

func (j *JwtMiddleware) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("Authorization")
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing token")
			}

			claims, err := j.JwtService.ValidateAccessToken(token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			c.Set("account_id", claims.AccountID)
			c.Set("organization_id", claims.OrganizationID)
			reqCtx := ctx.NewCtx(c)
			org, err := j.OrganizationService.GetOrganizationByID(reqCtx, claims.OrganizationID)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid organization")
			}

			account, err := j.AccountService.GetAccountByID(reqCtx, claims.AccountID)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid account")
			}

			reqCtx.SetOrg(org)
			reqCtx.SetUser(account)

			return next(c)
		}
	}
}
