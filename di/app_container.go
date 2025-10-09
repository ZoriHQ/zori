package di

import (
	"context"
	"fmt"

	_ "zori/docs" // Import generated swagger docs
	"zori/internal/config"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/internal/storage/postgres"
	"zori/services/auth"
	"zori/services/organizations"
	"zori/services/projects"

	"go.uber.org/fx"
)

func NewApplication() *fx.App {
	return fx.New(
		fx.Provide(
			config.NewConfig,
			postgres.NewPostgresDB,
			server.New,
		),

		auth.BuildAuthDIContainer(),
		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),

		fx.Provide(middlewares.NewJwtMiddleware),

		fx.Invoke(registerDatabaseLifecycle),
		fx.Invoke(server.RegisterSwaggerRoutes),

		projects.BuildProjectWebDIContainer(),
		organizations.BuildOrganizationWebDIContainer(),
		auth.BuildAuthWebDIContainer(),

		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						address := fmt.Sprintf("%s:%s", "0.0.0.0", "1323")
						fmt.Printf("Starting HTTP server on %s\n", address)
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

// registerDatabaseLifecycle registers database lifecycle hooks
func registerDatabaseLifecycle(lc fx.Lifecycle, db *postgres.PostgresDB) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fmt.Println("Database connection established")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("Closing database connection...")
			return db.Close()
		},
	})
}
