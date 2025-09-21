package postgres

import (
	"context"
	"embed"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Migrator struct {
	db       *bun.DB
	migrator *migrate.Migrator
}

func NewMigrator(db *bun.DB) *Migrator {
	migrations := migrate.NewMigrations()

	if err := migrations.Discover(migrationsFS); err != nil {
		panic(fmt.Sprintf("failed to discover migrations: %v", err))
	}

	migrator := migrate.NewMigrator(db, migrations)

	return &Migrator{
		db:       db,
		migrator: migrator,
	}
}

func (m *Migrator) Init(ctx context.Context) error {
	return m.migrator.Init(ctx)
}

func (m *Migrator) Migrate(ctx context.Context) error {
	group, err := m.migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if group.IsZero() {
		fmt.Println("No new migrations to run")
		return nil
	}

	fmt.Printf("Migrated to %s\n", group)
	return nil
}

func (m *Migrator) Rollback(ctx context.Context) error {
	group, err := m.migrator.Rollback(ctx)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	if group.IsZero() {
		fmt.Println("No migrations to rollback")
		return nil
	}

	fmt.Printf("Rolled back %s\n", group)
	return nil
}

func (m *Migrator) Status(ctx context.Context) error {
	ms, err := m.migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	fmt.Printf("%-20s %-10s\n", "Migration", "Status")
	fmt.Printf("%-20s %-10s\n", "=========", "======")

	for _, migration := range ms {
		status := "pending"
		if migration.GroupID > 0 {
			status = "applied"
		}
		fmt.Printf("%-20s %-10s\n", migration.Name, status)
	}

	return nil
}
