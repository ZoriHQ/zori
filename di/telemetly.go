package di

import (
	"context"
	"zori/internal/config"
	"zori/internal/logger"
	"zori/internal/telemetry"

	"go.uber.org/fx"
)

func NewTelemetryProvider(cfg *config.Config, lc fx.Lifecycle) (*telemetry.Provider, error) {
	provider, err := telemetry.NewProvider(telemetry.Config{
		ServiceName:           cfg.OTelServiceName + "-ingestion",
		ServiceVersion:        cfg.OTelServiceVersion,
		Environment:           cfg.OTelEnvironment,
		OTLPEndpoint:          cfg.OTelEndpoint,
		Enabled:               cfg.OTelEnabled,
		HTTPSamplingRate:      1.0,
		IngestionSamplingRate: cfg.OTelIngestSamplingRate,
	})
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return provider.Shutdown(ctx)
		},
	})

	return provider, nil
}

func NewLogger(cfg *config.Config) *logger.Logger {
	log := logger.NewLogger(logger.Config{
		Level:     cfg.LogLevel,
		Format:    cfg.LogFormat,
		LokiURL:   cfg.LokiURL,
		AddSource: true,
	})

	logger.SetDefault(log)
	return log
}
