package accesstokens

import (
	"zori/services/auth/services"
	"zori/services/auth/web"

	"go.uber.org/fx"
)

func BuildAccessTokensDIContainer() fx.Option {
	return fx.Module("auth",
		fx.Provide(
			services.NewTokenService,
			services.NewPasswordService,
			services.NewJWTService,
			services.NewAuthService,
		),
		fx.Invoke(web.RegisterRoutes),
	)
}
