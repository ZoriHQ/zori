package events

import (
	"context"
	"zori/services/events/services"
	"zori/services/events/web"

	"go.uber.org/fx"
)

func BuildEventsDIContainer() fx.Option {
	return fx.Module("events",
		fx.Provide(services.NewProcessor),
		fx.Provide(services.NewIdentifyProcessor),
		fx.Provide(services.NewEventsService),
		fx.Invoke(func(lc fx.Lifecycle, processorService *services.Processor, identifyProcessor *services.IdentifyProcessor) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go processorService.Start()
					go identifyProcessor.Start()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					if err := processorService.Stop(); err != nil {
						return err
					}
					return identifyProcessor.Stop()
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
