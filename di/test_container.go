package di

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"testing"
	"time"

	"zori/internal/cache"
	"zori/internal/config"
	"zori/internal/logger"
	"zori/internal/metrics"
	"zori/internal/natsstream"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/internal/storage/clickhouse"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"
	"zori/internal/telemetry"
	"zori/services/events"
	eventsServices "zori/services/events/services"
	"zori/services/ingestion"
	ingestionWeb "zori/services/ingestion/web"
	"zori/services/organizations"
	"zori/services/payments"
	paymentsServices "zori/services/payments/services"
	"zori/services/projects"
	"zori/services/revenue"
	revenueData "zori/services/revenue/data"
	revenueServices "zori/services/revenue/services"

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
	PaymentProcessor   *paymentsServices.PaymentProcessor
	RevenueService     *revenueServices.RevenueService
	IngestionServer    *ingestionWeb.IngestionServer
	IngestionServerURL string
	RevenueData        *revenueData.RevenueData
}

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
			func(cfg *config.Config) *telemetry.Provider {
				provider, _ := telemetry.NewProvider(telemetry.Config{
					ServiceName:           "zori-test",
					ServiceVersion:        "test",
					Environment:           "test",
					OTLPEndpoint:          cfg.OTelEndpoint,
					Enabled:               false,
					HTTPSamplingRate:      1.0,
					IngestionSamplingRate: 1.0,
				})
				return provider
			},
			func(cfg *config.Config) *logger.Logger {
				return logger.NewLogger(logger.Config{
					Level:  "error",
					Format: "json",
				})
			},
			clickhouse.NewClickhouseDB,
			natsstream.NewStream,
			cache.NewCacheService,
			server.New,
		),

		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),

		organizations.BuildOrganizationWebDIContainer(),
		projects.BuildProjectWebDIContainer(),

		ingestion.BuildIngestionDiContainer(),
		events.BuildEventsDIContainer(),

		payments.BuildPaymentsDIContainer(),
		revenue.BuildRevenueDIContainer(),

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

		fx.Invoke(func(lc fx.Lifecycle, processor *paymentsServices.PaymentProcessor) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go processor.Start()
					t.Logf("Started payment processor for testing")
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return processor.Stop()
				},
			})
		}),
		fx.NopLogger,
		fx.Populate(&tc.DB, &tc.ClickHouse, &tc.NATS, &tc.Server, &tc.Config, &tc.Cache, &tc.Processor, &tc.PaymentProcessor, &tc.RevenueService, &tc.IngestionServer, &tc.RevenueData),
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
