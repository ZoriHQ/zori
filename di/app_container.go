package di

import (
	"context"
	"fmt"

	_ "zori/docs" // Import generated swagger docs
	"zori/internal/cache"
	"zori/internal/config"
	ossInit "zori/internal/init"
	"zori/internal/natsstream"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/internal/storage/clickhouse"
	"zori/internal/storage/postgres"
	"zori/services/analytics"
	"zori/services/auth"
	"zori/services/events"
	"zori/services/organizations"
	orgServices "zori/services/organizations/services"
	"zori/services/payments"
	"zori/services/projects"
	"zori/services/revenue"

	"go.uber.org/fx"
)

func NewApplication() *fx.App {
	return fx.New(
		fx.Provide(
			config.NewConfig,
			postgres.NewPostgresDB,
			clickhouse.NewClickhouseDB,
			server.New,
		),
		fx.Provide(natsstream.NewStream),
		fx.Provide(cache.NewCacheService),
		fx.Provide(ossInit.NewOSSInitializer),
		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),
		analytics.BuildAnalyticsDIContainer(),
		revenue.BuildRevenueDIContainer(),
		payments.BuildPaymentsDIContainer(),

		// Conditional auth middleware provider
		fx.Provide(
			fx.Annotate(
				provideAuthMiddleware,
				fx.As(new(middlewares.AuthMiddleware)),
			),
		),
		fx.Provide(middlewares.NewCacheMiddleware),

		// Conditional auth service (only in OSS mode)
		fx.Invoke(func(cfg *config.Config) fx.Option {
			if cfg.ZoriOSS {
				return fx.Options(
					auth.BuildAuthDIContainer(),
					auth.BuildAuthWebDIContainer(),
				)
			}
			return fx.Options()
		}),

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
		payments.BuildPaymentsProcessorDIContainer(),
		payments.BuildPaymentsWebhookDIContainer(),

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

// provideAuthMiddleware conditionally provides the correct auth middleware
func provideAuthMiddleware(cfg *config.Config, db *postgres.PostgresDB, orgService *orgServices.OrganizationService) (middlewares.AuthMiddleware, error) {
	if cfg.ZoriOSS {
		return middlewares.NewOSSAuthMiddleware(cfg), nil
	}
	return middlewares.NewStackAuthMiddleware(cfg, orgService)
}

// registerDatabaseLifecycle registers database lifecycle hooks
func registerDatabaseLifecycle(lc fx.Lifecycle, db *postgres.PostgresDB) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("Closing database connection...")
			return db.Close()
		},
	})
}

// registerOSSInitializer runs OSS initialization if enabled
func registerOSSInitializer(lc fx.Lifecycle, initializer *ossInit.OSSInitializer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return initializer.Initialize(ctx)
		},
	})
}
