package di

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"zori/internal/cache"
	"zori/internal/config"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/internal/storage/clickhouse"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"
	"zori/internal/telemetry"
	"zori/services/analytics"
	analyticsData "zori/services/analytics/data"
	"zori/services/events"
	eventsServices "zori/services/events/services"
	"zori/services/ingestion"
	ingestionWeb "zori/services/ingestion/web"
	"zori/services/live"
	"zori/services/organizations"
	"zori/services/projects"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/valyala/fasthttp"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

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
	AnalyticsData      *analyticsData.AnalyticsData
}

var (
	sharedContainer     *TestContainer
	sharedContainerOnce sync.Once
	sharedContainerMu   sync.Mutex
)

func NewTestPostgresDB(cfg *config.Config) (*postgres.PostgresDB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.PostgresURL)))
	db := bun.NewDB(sqldb, pgdialect.New())

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

	serverReady := make(chan string, 1)

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
			telemetry.NewLogger,
			telemetry.NewTracerProvider,
		),

		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),

		organizations.BuildOrganizationWebDIContainer(),
		projects.BuildProjectWebDIContainer(),

		ingestion.BuildIngestionDiContainer(),
		live.BuildLiveDIContainer(),
		events.BuildEventsDIContainer(),
		analytics.BuildAnalyticsDIContainer(),

		fx.Provide(metrics.NewMetricsCollector),
		fx.Provide(metrics.NewIngestMetrics),
		fx.Provide(metrics.NewNatsMetrics),
		fx.Provide(metrics.NewMetricsServer),

		fx.Provide(
			fx.Annotate(
				provideAuthMiddleware,
				fx.As(new(middlewares.AuthMiddleware)),
			),
		),

		fx.Invoke(func(lc fx.Lifecycle, ingestionServer *ingestionWeb.IngestionServer) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						ln, err := net.Listen("tcp", "localhost:0")
						if err != nil {
							t.Logf("Failed to create listener: %v", err)
							serverReady <- ""
							return
						}

						addr := ln.Addr().String()
						t.Logf("Starting test ingestion server on %s", addr)
						serverReady <- fmt.Sprintf("http://%s", addr)

						if err := fasthttp.Serve(ln, ingestionServer.HandleRequest); err != nil {
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

		fx.NopLogger,
		fx.Populate(&tc.DB, &tc.ClickHouse, &tc.NATS, &tc.Server, &tc.Config, &tc.Cache, &tc.Processor, &tc.IngestionServer, &tc.AnalyticsData),
	)

	tc.App = app
	app.RequireStart()

	select {
	case url := <-serverReady:
		if url == "" {
			t.Fatal("Failed to start ingestion server")
		}
		tc.IngestionServerURL = url
		t.Logf("Ingestion server ready at %s", url)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for ingestion server to start")
	}

	return tc
}

func (tc *TestContainer) Cleanup() {
	tc.App.RequireStop()
}

func GetSharedTestContainer(t *testing.T) *TestContainer {
	sharedContainerOnce.Do(func() {
		sharedContainer = NewTestContainer(t)
	})
	return sharedContainer
}

func (tc *TestContainer) CleanupTestData(ctx context.Context) error {
	sharedContainerMu.Lock()
	defer sharedContainerMu.Unlock()

	if _, err := tc.DB.DB.NewTruncateTable().
		Model((*models.Project)(nil)).
		Cascade().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to truncate projects: %w", err)
	}

	if _, err := tc.DB.DB.NewTruncateTable().
		Model((*models.Organization)(nil)).
		Cascade().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to truncate organizations: %w", err)
	}

	if _, err := tc.DB.DB.NewTruncateTable().
		Model((*models.Account)(nil)).
		Cascade().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to truncate accounts: %w", err)
	}

	if err := tc.ClickHouse.Db().Exec(ctx, "TRUNCATE TABLE IF EXISTS events"); err != nil {
		return fmt.Errorf("failed to truncate events: %w", err)
	}

	if err := tc.Cache.FlushAll(ctx); err != nil {
		return fmt.Errorf("failed to flush cache: %w", err)
	}

	return nil
}
