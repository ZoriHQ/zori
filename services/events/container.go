package events

import (
	"context"
	"zori/services/events/services"
	"zori/services/events/web"

	"go.uber.org/fx"
)

func BuildEventsDIContainer() fx.Option {
	return fx.Module("events",
		fx.Provide(services.NewBatchProcessor),
		fx.Provide(services.NewProcessor),
		fx.Provide(services.NewIdentifyProcessor),
		fx.Provide(services.NewEventsService),
		fx.Provide(services.NewJWTService),
		fx.Invoke(func(lc fx.Lifecycle, batchProcessor *services.BatchProcessor, processorService *services.Processor, identifyProcessor *services.IdentifyProcessor) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go batchProcessor.Start()
					go processorService.Start()
					go identifyProcessor.Start()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					if err := identifyProcessor.Stop(); err != nil {
						return err
					}
					if err := processorService.Stop(); err != nil {
						return err
					}
					return batchProcessor.Stop()
				},
			})
		}),
	)
}

func BuildEventsWebContainer() fx.Option {
	return fx.Module("events.web",
		fx.Invoke(web.RegisterRoutes),
	)
}
