package di

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"zori/internal/config"
	"zori/internal/server"
	"zori/internal/server/middlewares"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"
	"zori/services/auth"
	"zori/services/organizations"
	"zori/services/projects"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// TestContainer holds all the dependencies needed for testing
type TestContainer struct {
	App    *fxtest.App
	DB     *postgres.PostgresDB
	Server *server.Server
	Config *config.Config
}

func NewTestConfig() *config.Config {
	testPostgresURL := os.Getenv("TEST_POSTGRES_URL")
	if testPostgresURL == "" {
		testPostgresURL = "postgres://postgres:postgres@localhost:5432/zori_test?sslmode=disable"
	}

	testClickHouseURL := os.Getenv("TEST_CLICKHOUSE_URL")
	if testClickHouseURL == "" {
		testClickHouseURL = "clickhouse://localhost:9000/zori_test"
	}

	return &config.Config{
		PostgresURL:        testPostgresURL,
		ClickHouseURL:      testClickHouseURL,
		JWTSecretKey:       "test-secret-key-for-testing-purposes-min-32-chars",
		JWTAccessTokenTTL:  15 * time.Minute,
		JWTRefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:         4,
	}
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

	app := fxtest.New(
		t,
		fx.Provide(
			NewTestConfig,
			func(cfg *config.Config) (*postgres.PostgresDB, error) {
				return NewTestPostgresDB(cfg)
			},
			server.New,
		),

		auth.BuildAuthDIContainer(),
		organizations.BuildOrganizationDIContainer(),
		projects.BuildProjectsDIContainer(),

		// Jwt middleware must be provided after the auth & org containers are built since it depends on some of the auth services
		fx.Provide(middlewares.NewJwtMiddleware),

		fx.Populate(&tc.DB, &tc.Server, &tc.Config),
	)

	tc.App = app
	app.RequireStart()

	return tc
}

func (tc *TestContainer) Cleanup() {
	tc.App.RequireStop()
}
