package di

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"marker/internal/config"
	"marker/internal/server"
	"marker/internal/storage/postgres"
	"marker/services/auth"
	"marker/services/auth/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// Note: We'll use goose without embedded migrations, letting it read from disk
// since go:embed doesn't support parent directory access

// TestContainer holds all the dependencies needed for testing
type TestContainer struct {
	App    *fxtest.App
	DB     *postgres.PostgresDB
	Server *server.Server
	Config *config.Config
}

// TestConfig creates a test configuration
func NewTestConfig() *config.Config {
	// Use test-specific environment variables or defaults
	testPostgresURL := os.Getenv("TEST_POSTGRES_URL")
	if testPostgresURL == "" {
		testPostgresURL = "postgres://postgres:postgres@localhost:5432/marker_test?sslmode=disable"
	}

	testClickHouseURL := os.Getenv("TEST_CLICKHOUSE_URL")
	if testClickHouseURL == "" {
		testClickHouseURL = "clickhouse://localhost:9000/marker_test"
	}

	return &config.Config{
		PostgresURL:        testPostgresURL,
		ClickHouseURL:      testClickHouseURL,
		JWTSecretKey:       "test-secret-key-for-testing-purposes-min-32-chars",
		JWTAccessTokenTTL:  15 * time.Minute,
		JWTRefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:         4, // Lower cost for faster tests
	}
}

// NewTestPostgresDB creates a test database connection
func NewTestPostgresDB(cfg *config.Config) (*postgres.PostgresDB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.PostgresURL)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Test the connection
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping test database: %w", err)
	}

	// Register models for many-to-many relationships
	// IMPORTANT: Register the intermediate table first, before the models that reference it
	db.RegisterModel((*models.OrganizationMember)(nil))
	db.RegisterModel(
		(*models.Account)(nil),
		(*models.Organization)(nil),
	)

	return &postgres.PostgresDB{DB: db}, nil
}

// NewTestContainer creates a new test container with all dependencies
func NewTestContainer(t *testing.T) *TestContainer {
	tc := &TestContainer{}

	app := fxtest.New(
		t,
		// Core providers
		fx.Provide(
			NewTestConfig,
			func(cfg *config.Config) (*postgres.PostgresDB, error) {
				return NewTestPostgresDB(cfg)
			},
			server.New,
		),

		// Auth module
		auth.BuildAuthDIContainer(),

		// Populate the test container
		fx.Populate(&tc.DB, &tc.Server, &tc.Config),
	)

	tc.App = app
	app.RequireStart()

	return tc
}

// Cleanup cleans up the test container
func (tc *TestContainer) Cleanup() {
	tc.App.RequireStop()
}
