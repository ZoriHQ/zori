package live

import (
	"zori/services/live/services"
	"zori/services/live/web"

	"go.uber.org/fx"
)

func BuildLiveDIContainer() fx.Option {
	return fx.Module("live",
		fx.Provide(services.NewLiveSessionTracker),
	)
}

func BuildLiveWebDIContainer() fx.Option {
	return fx.Module("live_web",
		fx.Invoke(web.RegisterRoutes),
	)
}
