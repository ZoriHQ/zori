package organizations

import (
	"zori/services/organizations/data"
	"zori/services/organizations/services"
	"zori/services/organizations/web"

	"go.uber.org/fx"
)

func BuildOrganizationDIContainer() fx.Option {
	return fx.Module("organizatioon",
		fx.Provide(
			data.NewAccountData,
			data.NewOrganizationData,
			services.NewOrganizationService,
			services.NewAccountService,
		),
	)
}

func BuildOrganizationWebDIContainer() fx.Option {
	return fx.Module("organization_web",
		fx.Invoke(web.RegisterRoutes),
	)
}
