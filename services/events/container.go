package events

import (
	"context"
	"zori/services/events/services"

	"go.uber.org/fx"
)

func BuildEventsDIContainer() fx.Option {
	return fx.Module("events",
		fx.Provide(services.NewProcessor),
		fx.Invoke(func(lc fx.Lifecycle, processorService *services.Processor) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go processorService.Start()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return processorService.Stop()
				},
			})
		}),
	)
}
