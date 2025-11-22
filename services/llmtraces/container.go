package llmtraces

import (
	"context"

	"zori/services/llmtraces/data"
	"zori/services/llmtraces/services"
	"zori/services/llmtraces/tiles"
	"zori/services/llmtraces/web"

	"go.uber.org/fx"
)

func BuildLLMTracesDIContainer() fx.Option {
	return fx.Module("llmtraces",
		fx.Provide(data.NewLLMTraceRepository),
		fx.Provide(services.NewTokenCounter),
		fx.Provide(services.NewPriceFetcher),
		fx.Provide(services.NewLLMTraceIngestor),
		fx.Provide(services.NewLLMTraceProcessor),
		fx.Provide(services.NewLLMTracesService),
		fx.Provide(tiles.NewAvgPricePerCustomerTile),
		fx.Provide(tiles.NewDailyPriceTile),
		fx.Provide(tiles.NewDailyPriceTimelineTile),
		fx.Provide(services.NewLLMTracesTilesService),
	)
}

func BuildLLMTracesWebDIContainer() fx.Option {
	return fx.Module("llmtraces_web",
		fx.Invoke(web.RegisterRoutes),
	)
}

func BuildLLMTracesProcessorDIContainer() fx.Option {
	return fx.Module("llmtraces_processor",
		fx.Invoke(func(lc fx.Lifecycle, processor *services.LLMTraceProcessor) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return processor.Start(ctx)
				},
				OnStop: func(ctx context.Context) error {
					return processor.Stop()
				},
			})
		}),
	)
}
