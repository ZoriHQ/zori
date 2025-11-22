package funnels

import (
	"zori/services/funnels/data"
	"zori/services/funnels/services"
	"zori/services/funnels/web"

	"go.uber.org/fx"
)

func BuildFunnelsDIContainer() fx.Option {
	return fx.Module("funnels",
		fx.Provide(data.NewFunnelRepository),
		fx.Provide(data.NewFunnelAnalyticsData),
		fx.Provide(services.NewFunnelService),
	)
}

func BuildFunnelsWebDIContainer() fx.Option {
	return fx.Module("funnels_web",
		fx.Invoke(web.RegisterRoutes),
	)
}
