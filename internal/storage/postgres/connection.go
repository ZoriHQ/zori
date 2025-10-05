package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"marker/internal/config"
	"marker/internal/storage/postgres/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type PostgresDB struct {
	*bun.DB
}

func NewPostgresDB(cfg *config.Config) *PostgresDB {
	if cfg.PostgresURL == "" {
		panic("POSTGRES_URL is required")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.PostgresURL)))

	db := bun.NewDB(sqldb, pgdialect.New())

	// Add query hook for debugging in development
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	if err := db.PingContext(context.Background()); err != nil {
		panic(fmt.Errorf("failed to ping database: %w", err))
	}

	db.RegisterModel((*models.OrganizationMember)(nil))
	db.RegisterModel(
		(*models.Account)(nil),
		(*models.Organization)(nil),
	)

	return &PostgresDB{DB: db}
}

func (p *PostgresDB) Db() *bun.DB {
	return p.DB
}

func (p *PostgresDB) Close() error {
	return p.DB.Close()
}

func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}
