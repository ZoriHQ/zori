package events

import (
	"context"
	"zori/internal/telemetry"
	"zori/services/events/services"
	"zori/services/events/web"
	"zori/services/ingestion/data"
	liveServices "zori/services/live/services"

	"go.uber.org/fx"
)

func BuildEventsDIContainer() fx.Option {
	return fx.Module("events",
		fx.Provide(services.NewBatchProcessor),
		fx.Provide(fx.Private, data.NewVisitorRepository),
		fx.Provide(services.NewProcessor),
		fx.Provide(services.NewIdentifyProcessor),
		fx.Provide(services.NewEventsService),
		fx.Provide(services.NewJWTService),
		fx.Invoke(func(lc fx.Lifecycle, batchProcessor *services.BatchProcessor, processorService *services.Processor, identifyProcessor *services.IdentifyProcessor, liveTracker *liveServices.LiveSessionTracker, logger *telemetry.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// Start the live session tracker first
					if err := liveTracker.Start(); err != nil {
						logger.Fatal("Live session tracker failed to start", telemetry.Error(err))
					}
					go func() {
						if err := batchProcessor.Start(); err != nil {
							logger.Fatal("Batch processor failed", telemetry.Error(err))
						}
					}()
					go func() {
						if err := processorService.Start(); err != nil {
							logger.Fatal("Event processor failed", telemetry.Error(err))
						}
					}()
					go func() {
						if err := identifyProcessor.Start(); err != nil {
							logger.Fatal("Identify processor failed", telemetry.Error(err))
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					if err := identifyProcessor.Stop(); err != nil {
						return err
					}
					if err := processorService.Stop(); err != nil {
						return err
					}
					if err := liveTracker.Stop(); err != nil {
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
