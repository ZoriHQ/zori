package organizations

import (
	"marker/services/organizations/data"
	"marker/services/organizations/services"
	"marker/services/organizations/web"

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
		fx.Invoke(web.RegisterRoutes),
	)
}
