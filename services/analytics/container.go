package analytics

import (
	"zori/services/analytics/data"
	"zori/services/analytics/services"
	"zori/services/analytics/tiles"
	"zori/services/analytics/web"
	ingestionData "zori/services/ingestion/data"

	"go.uber.org/fx"
)

func BuildAnalyticsDIContainer() fx.Option {
	return fx.Module("analytics",
		fx.Provide(ingestionData.NewVisitorRepository),
		fx.Provide(data.NewAnalyticsData),

		fx.Provide(tiles.NewTimelineTile),
		fx.Provide(tiles.NewTrafficSourceTile),

		fx.Provide(services.NewAnalyticsService),
		fx.Provide(services.NewVisitorsService),
		fx.Provide(services.NewTilesService),
	)
}

func BuildAnalyticsWebDIContainer() fx.Option {
	return fx.Module("analytics_web",
		fx.Invoke(web.RegisterRoutes),
	)
}
