package revenue

import (
	ingestionData "zori/services/ingestion/data"
	"zori/services/revenue/data"
	"zori/services/revenue/services"
	"zori/services/revenue/web"

	"go.uber.org/fx"
)

func BuildRevenueDIContainer() fx.Option {
	return fx.Module("revenue",
		fx.Provide(fx.Private, ingestionData.NewVisitorRepository),
		fx.Provide(data.NewRevenueData),
		fx.Provide(services.NewRevenueService),
	)
}

func BuildRevenueWebDIContainer() fx.Option {
	return fx.Module("revenue_web",
		fx.Invoke(web.RegisterRoutes),
	)
}
