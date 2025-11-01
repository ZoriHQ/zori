package di

import (
	"context"
	"fmt"
	_ "zori/docs" // Import generated swagger docs
	"zori/internal/cache"
	"zori/internal/config"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/server"
	"zori/internal/storage/postgres"
	"zori/services/ingestion"
	"zori/services/ingestion/web"
	"zori/services/organizations"
	"zori/services/payments"
	"zori/services/projects"

	"github.com/valyala/fasthttp"
	"go.uber.org/fx"
)

func NewIngestionApplication() *fx.App {
	return fx.New(
		fx.Provide(
			config.NewConfig,
			postgres.NewPostgresDB,
			server.New,
		),

		fx.Provide(natsstream.NewStream),
		fx.Provide(cache.NewCacheService),

		fx.Provide(metrics.NewMetricsCollector),
		fx.Provide(metrics.NewIngestMetrics),
		fx.Provide(metrics.NewNatsMetrics),
		fx.Provide(metrics.NewMetricsServer),

		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),
		payments.BuildPaymentsDIContainer(),

		fx.Invoke(registerDatabaseLifecycle),
		ingestion.BuildIngestionDiContainer(),

		fx.Invoke(func(lc fx.Lifecycle, metricsServer *metrics.MetricsServer) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return metricsServer.Start()
				},
				OnStop: func(ctx context.Context) error {
					return metricsServer.Stop(ctx)
				},
			})
		}),

		fx.Invoke(func(lc fx.Lifecycle, ingestionServer *web.IngestionServer) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						address := fmt.Sprintf("%s:%s", "0.0.0.0", "1324")
						fmt.Printf("Starting Ingestion server on %s\n", address)
						if err := fasthttp.ListenAndServe(address, ingestionServer.HandleRequest); err != nil {
							fmt.Printf("Server error: %v\n", err)
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					fmt.Println("Shutting down Ingestion server...")
					return nil
				},
			})
		}),

		fx.NopLogger,
	)
}
