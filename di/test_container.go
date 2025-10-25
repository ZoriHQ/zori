package di

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"zori/internal/cache"
	"zori/internal/config"
	"zori/internal/natsstream"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/internal/storage/clickhouse"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"
	"zori/services/auth"
	"zori/services/events"
	eventsServices "zori/services/events/services"
	"zori/services/ingestion"
	ingestionWeb "zori/services/ingestion/web"
	"zori/services/organizations"
	"zori/services/projects"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/valyala/fasthttp"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// TestContainer holds all the dependencies needed for testing
type TestContainer struct {
	App                *fxtest.App
	DB                 *postgres.PostgresDB
	ClickHouse         *clickhouse.ClickhouseDB
	NATS               *natsstream.Stream
	Server             *server.Server
	Config             *config.Config
	Cache              *cache.CacheService
	Processor          *eventsServices.Processor
	IngestionServer    *ingestionWeb.IngestionServer
	IngestionServerURL string
}

func NewTestPostgresDB(cfg *config.Config) (*postgres.PostgresDB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.PostgresURL)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Test the connection
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping test database: %w", err)
	}

	db.RegisterModel((*models.OrganizationMember)(nil))
	db.RegisterModel(
		(*models.Account)(nil),
		(*models.Organization)(nil),
		(*models.Project)(nil),
	)

	return &postgres.PostgresDB{DB: db}, nil
}

func NewTestContainer(t *testing.T) *TestContainer {
	tc := &TestContainer{}
	tc.IngestionServerURL = "http://localhost:19324"

	app := fxtest.New(
		t,
		fx.Provide(
			config.NewConfig,
			func(cfg *config.Config) (*postgres.PostgresDB, error) {
				return NewTestPostgresDB(cfg)
			},
			clickhouse.NewClickhouseDB,
			natsstream.NewStream,
			cache.NewCacheService,
			server.New,
		),

		auth.BuildAuthDIContainer(),
		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),

		// Jwt middleware must be provided after the auth & org containers are built since it depends on some of the auth services
		fx.Provide(middlewares.NewJwtMiddleware),

		// Register web routes for testing
		auth.BuildAuthWebDIContainer(),
		organizations.BuildOrganizationWebDIContainer(),
		projects.BuildProjectWebDIContainer(),

		// Add ingestion and events modules
		ingestion.BuildIngestionDiContainer(),
		events.BuildEventsDIContainer(),

		// Start fasthttp ingestion server for testing
		fx.Invoke(func(lc fx.Lifecycle, ingestionServer *ingestionWeb.IngestionServer) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						address := "localhost:19324"
						t.Logf("Starting test ingestion server on %s", address)
						if err := fasthttp.ListenAndServe(address, ingestionServer.HandleRequest); err != nil {
							t.Logf("Ingestion server error: %v", err)
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return nil
				},
			})
		}),

		fx.Populate(&tc.DB, &tc.ClickHouse, &tc.NATS, &tc.Server, &tc.Config, &tc.Cache, &tc.Processor, &tc.IngestionServer),
	)

	tc.App = app
	app.RequireStart()

	return tc
}

func (tc *TestContainer) Cleanup() {
	tc.App.RequireStop()
}
