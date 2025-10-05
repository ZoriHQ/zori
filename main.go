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

// @title           OpenAPI Specification for Zori server
// @version         1.0
// @termsOfService  https://swagger.io/terms/

// @contact.name   Zori Support
// @contact.url    https://www.zorihq.com/support
// @contact.email  support@zorihq.com

// @license.name  Apache 2.0
// @license.url   https://www.apache.org/licenses/LICENSE-2.0.html

// @host      api.prod.zorihq.com
// @BasePath  /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

// @externalDocs.description  Zori API Documentation
// @externalDocs.url          https://docs.zorihq.com
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
