package recordings

import (
	"context"
	"zori/internal/telemetry"
	"zori/services/recordings/data"
	"zori/services/recordings/services"
	"zori/services/recordings/web"

	"go.uber.org/fx"
)

func BuildRecordingsDIContainer() fx.Option {
	return fx.Module("recordings",
		fx.Provide(services.NewRecordingBatchProcessor),
		fx.Provide(services.NewRecordingProcessor),
		fx.Provide(data.NewRecordingsData),
		fx.Provide(services.NewRecordingsService),
		fx.Invoke(func(lc fx.Lifecycle, batchProcessor *services.RecordingBatchProcessor, processor *services.RecordingProcessor, logger *telemetry.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						if err := batchProcessor.Start(); err != nil {
							logger.Fatal("Recording batch processor failed", telemetry.Error(err))
						}
					}()
					go func() {
						if err := processor.Start(); err != nil {
							logger.Fatal("Recording processor failed", telemetry.Error(err))
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					if err := processor.Stop(); err != nil {
						return err
					}
					return batchProcessor.Stop()
				},
			})
		}),
	)
}

func BuildRecordingsWebContainer() fx.Option {
	return fx.Module("recordings.web",
		fx.Invoke(web.RegisterRoutes),
	)
}
