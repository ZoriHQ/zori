package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"marker/internal/config"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type PostgresDB struct {
	*bun.DB
}

func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	if cfg.PostgresURL == "" {
		return nil, fmt.Errorf("POSTGRES_URL is required")
	}

	// Create a database/sql DB connection
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.PostgresURL)))

	// Create a Bun DB instance
	db := bun.NewDB(sqldb, pgdialect.New())

	// Add query hook for debugging in development
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	// Test the connection
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{DB: db}, nil
}

func (p *PostgresDB) Close() error {
	return p.DB.Close()
}

func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}
