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
		fx.Provide(tiles.NewTrafficRefererSourceTile),
		fx.Provide(tiles.NewTrafficCountrySourceTile),
		fx.Provide(tiles.NewTrafficUTMSourceTile),
		fx.Provide(tiles.NewUniqueVisitorsTile),
		fx.Provide(tiles.NewUniqueSessionsTile),
		fx.Provide(tiles.NewBounceRateTile),
		fx.Provide(tiles.NewSessionDurationTile),
		fx.Provide(tiles.NewPagesPerSessionTile),
		fx.Provide(tiles.NewDAUTile),
		fx.Provide(tiles.NewWAUTile),
		fx.Provide(tiles.NewMAUTile),
		fx.Provide(tiles.NewReturnRateTile),
		fx.Provide(tiles.NewTimeBetweenVisitsTile),

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
