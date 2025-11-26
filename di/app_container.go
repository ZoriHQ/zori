package di

import (
	"context"
	"fmt"
	"os"

	_ "zori/docs" // Import generated swagger docs
	"zori/internal/cache"
	"zori/internal/config"
	ossInit "zori/internal/init"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/internal/storage/clickhouse"
	"zori/internal/storage/postgres"
	"zori/internal/telemetry"
	"zori/services/analytics"
	"zori/services/auth"
	"zori/services/events"
	"zori/services/live"
	"zori/services/organizations"
	orgServices "zori/services/organizations/services"
	"zori/services/payments"
	"zori/services/projects"
	"zori/services/revenue"

	"go.uber.org/fx"
)

func provideForOSS() fx.Option {
	if os.Getenv("ZORI_IS_OSS") == "true" {
		return auth.BuildAuthDIContainer()
	}
	return fx.Options()
}

func buildForOSS() fx.Option {
	if os.Getenv("ZORI_IS_OSS") == "true" {
		fmt.Println("Building OSS")
		return auth.BuildAuthWebDIContainer()
	}
	return fx.Options()
}

func NewApplication() *fx.App {
	return fx.New(
		fx.Provide(
			config.NewConfig,
			postgres.NewPostgresDB,
			clickhouse.NewClickhouseDB,
			server.New,
			telemetry.NewLogger,
			telemetry.NewTracerProvider,
		),
		fx.Provide(natsstream.NewStream),
		fx.Provide(cache.NewCacheService),
		fx.Provide(ossInit.NewOSSInitializer),

		fx.Provide(metrics.NewMetricsCollector),
		fx.Provide(metrics.NewIngestMetrics),
		fx.Provide(metrics.NewNatsMetrics),
		fx.Provide(metrics.NewMetricsServer),

		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),
		analytics.BuildAnalyticsDIContainer(),
		revenue.BuildRevenueDIContainer(),
		payments.BuildPaymentsDIContainer(),
		live.BuildLiveDIContainer(),

		fx.Provide(
			fx.Annotate(
				provideAuthMiddleware,
				fx.As(new(middlewares.AuthMiddleware)),
			),
		),
		fx.Provide(middlewares.NewCacheMiddleware),
		provideForOSS(),
		buildForOSS(),

		fx.Invoke(registerDatabaseLifecycle),
		fx.Invoke(registerOSSInitializer),
		fx.Invoke(server.RegisterSwaggerRoutes),

		projects.BuildProjectWebDIContainer(),
		organizations.BuildOrganizationWebDIContainer(),
		analytics.BuildAnalyticsWebDIContainer(),
		revenue.BuildRevenueWebDIContainer(),
		payments.BuildPaymentsWebDIContainer(),
		events.BuildEventsDIContainer(),
		events.BuildEventsWebContainer(),
		live.BuildLiveWebDIContainer(),
		payments.BuildPaymentsProcessorDIContainer(),
		payments.BuildPaymentsWebhookDIContainer(),

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

		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						address := fmt.Sprintf("%s:%s", "0.0.0.0", "1323")
						if err := srv.Echo.Start(address); err != nil {
							fmt.Printf("Server error: %v\n", err)
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					fmt.Println("Shutting down HTTP server...")
					return srv.Echo.Shutdown(ctx)
				},
			})
		}),

		fx.NopLogger,
	)
}

func provideAuthMiddleware(cfg *config.Config, db *postgres.PostgresDB, orgService *orgServices.OrganizationService) (middlewares.AuthMiddleware, error) {
	if cfg.ZoriOSS {
		return middlewares.NewOSSAuthMiddleware(cfg), nil
	}
	return middlewares.NewClerkAuthMiddleware(cfg, orgService)
}

func registerDatabaseLifecycle(lc fx.Lifecycle, db *postgres.PostgresDB) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})
}

func registerOSSInitializer(lc fx.Lifecycle, initializer *ossInit.OSSInitializer, cfg *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			resetAuth := false
			if val := ctx.Value(config.ResetAuthKey); val != nil {
				if b, ok := val.(bool); ok {
					resetAuth = b
				}
			}
			return initializer.InitializeOrReset(ctx, resetAuth)
		},
	})
}
