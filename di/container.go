package di

import (
	"context"
	"fmt"

	"marker/internal/config"
	"marker/internal/server"
	"marker/internal/storage/postgres"
	"marker/services/auth"

	"go.uber.org/fx"
)

func NewApplication() *fx.App {
	return fx.New(
		// Core providers
		fx.Provide(
			config.NewConfig,
			postgres.NewPostgresDB,
			postgres.NewMigrator,
			server.New,
		),

		auth.BuildAuthDIContainer(),

		fx.Invoke(registerDatabaseLifecycle),

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

func RunMigrations(action string) error {
	app := fx.New(
		fx.Provide(
			config.NewConfig,
			postgres.NewPostgresDB,
			postgres.NewMigrator,
		),
		fx.Invoke(func(migrator *postgres.Migrator) error {
			ctx := context.Background()

			switch action {
			case "up":
				fmt.Println("Running migrations...")
				return migrator.Migrate(ctx)
			case "down":
				fmt.Println("Rolling back migration...")
				return migrator.Rollback(ctx)
			case "status":
				fmt.Println("Migration status:")
				return migrator.Status(ctx)
			case "init":
				fmt.Println("Initializing migrations...")
				return migrator.Init(ctx)
			default:
				return fmt.Errorf("unknown migration action: %s", action)
			}
		}),
		fx.NopLogger,
	)

	return app.Start(context.Background())
}
