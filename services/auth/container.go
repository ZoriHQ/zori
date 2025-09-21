package auth

import (
	"marker/services/auth/services"

	"go.uber.org/fx"
)

func BuildAuthDIContainer() fx.Option {
	return fx.Module("auth",
		fx.Provide(
			services.NewTokenService,
			services.NewPasswordService,
			services.NewJWTService,
			services.NewAuthService,
		),
	)
}
