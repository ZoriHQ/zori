package auth

import (
	"fmt"
	"zori/services/auth/data"
	"zori/services/auth/services"
	"zori/services/auth/web"

	"go.uber.org/fx"
)

func BuildAuthDIContainer() fx.Option {
	return fx.Module("auth",
		fx.Provide(
			data.NewAuthData,
			services.NewAuthService,
		),
	)
}

func BuildAuthWebDIContainer() fx.Option {
	fmt.Println("Registering")
	return fx.Module("auth_web",
		fx.Invoke(web.RegisterRoutes),
	)
}
