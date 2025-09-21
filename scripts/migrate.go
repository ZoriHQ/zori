package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"marker/internal/config"
	"marker/internal/storage/postgres"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/migrate.go <migrate|rollback|status>")
		os.Exit(1)
	}

	action := os.Args[1]
	cfg := config.NewConfig()
	ctx := context.Background()

	fmt.Printf("Connecting to PostgreSQL...\n")

	db, err := postgres.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	migrator := postgres.NewMigrator(db.DB)

	switch action {
	case "migrate":
		fmt.Println("Running auth table migrations...")
		if err := migrator.Init(ctx); err != nil {
			log.Fatalf("Failed to initialize migrations: %v", err)
		}
		if err := migrator.Migrate(ctx); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations completed successfully!")

	case "rollback":
		fmt.Println("Rolling back migrations...")
		if err := migrator.Rollback(ctx); err != nil {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		fmt.Println("Rollback completed successfully!")

	case "status":
		fmt.Println("Migration status:")
		if err := migrator.Status(ctx); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}

	default:
		fmt.Printf("Unknown action: %s\n", action)
		fmt.Println("Available actions: migrate, rollback, status")
		os.Exit(1)
	}
}
