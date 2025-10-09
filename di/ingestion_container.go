package di

import (
	"context"
	"fmt"
	_ "zori/docs" // Import generated swagger docs
	"zori/internal/config"
	"zori/internal/natsstream"
	"zori/internal/server"
	"zori/internal/storage/postgres"
	"zori/services/auth"
	"zori/services/ingestion"
	"zori/services/ingestion/web"
	"zori/services/organizations"
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

		auth.BuildAuthDIContainer(),
		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),

		fx.Invoke(registerDatabaseLifecycle),
		ingestion.BuildIngestionDiContainer(),

		fx.Invoke(func(lc fx.Lifecycle, ingestionServer *web.IngestionServer) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						address := fmt.Sprintf("%s:%s", "0.0.0.0", "1324")
						fmt.Printf("Starting Ingestion server on %s\n", address)
						if err := fasthttp.ListenAndServe(address, ingestionServer.Injest); err != nil {
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
