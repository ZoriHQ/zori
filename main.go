package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marker/di"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "marker",
		Usage: "Marker application with server and migration commands",
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Start the HTTP server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Value:   "8080",
						Usage:   "Port to run server on",
					},
					&cli.StringFlag{
						Name:    "host",
						Aliases: []string{"H"},
						Value:   "0.0.0.0",
						Usage:   "Host to bind server to",
					},
				},
				Action: runServer,
			},
			{
				Name:  "migrate",
				Usage: "Database migration commands",
				Commands: []*cli.Command{
					{
						Name:   "up",
						Usage:  "Run all pending migrations",
						Action: migrateUp,
					},
					{
						Name:   "down",
						Usage:  "Rollback the last migration",
						Action: migrateDown,
					},
					{
						Name:   "status",
						Usage:  "Show migration status",
						Action: migrateStatus,
					},
					{
						Name:   "init",
						Usage:  "Initialize migration tables",
						Action: migrateInit,
					},
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(ctx context.Context, cmd *cli.Command) error {
	app := di.NewApplication()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	startCtx, startCancel := context.WithCancel(ctx)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	<-ctx.Done()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	fmt.Println("Shutting down application...")
	if err := app.Stop(stopCtx); err != nil {
		return fmt.Errorf("failed to stop application gracefully: %w", err)
	}

	fmt.Println("Application stopped successfully")
	return nil
}

func migrateUp(ctx context.Context, cmd *cli.Command) error {
	return di.RunMigrations("up")
}

func migrateDown(ctx context.Context, cmd *cli.Command) error {
	return di.RunMigrations("down")
}

func migrateStatus(ctx context.Context, cmd *cli.Command) error {
	return di.RunMigrations("status")
}

func migrateInit(ctx context.Context, cmd *cli.Command) error {
	return di.RunMigrations("init")
}
