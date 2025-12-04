package traces

import (
	"zori/services/traces/data"
	"zori/services/traces/services"
	"zori/services/traces/web"

	"go.uber.org/fx"
)

func BuildTracesDIContainer() fx.Option {
	return fx.Module("traces",
		fx.Provide(data.NewTracesRepository),
		fx.Provide(services.NewIngestionService),
		fx.Provide(web.NewTracesHandler),
	)
}
