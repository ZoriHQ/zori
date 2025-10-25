// Dagger CI module for Zori
//
// This module provides CI/CD pipelines for the Zori project.
// It handles integration testing with PostgreSQL, ClickHouse, NATS, and Redis.

package main

import (
	"context"
	"fmt"

	"dagger/zori/internal/dagger"
)

type Zori struct{}

// Test runs the full integration test suite with all required services
func (m *Zori) Test(ctx context.Context, source *dagger.Directory) (string, error) {
	// Start required services
	postgres := m.postgres()
	clickhouse := m.clickhouse()
	nats := m.nats()
	redis := m.redis()

	// Build the test container with all dependencies
	container := m.testContainer(source).
		WithServiceBinding("postgres", postgres).
		WithServiceBinding("clickhouse", clickhouse).
		WithServiceBinding("nats", nats).
		WithServiceBinding("redis", redis).
		WithEnvVariable("POSTGRES_URL", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable").
		WithEnvVariable("CLICKHOUSE_URL", "clickhouse://default:default@clickhouse:9000").
		WithEnvVariable("CLICKHOUSE_USERNAME", "default").
		WithEnvVariable("CLICKHOUSE_USER", "default").
		WithEnvVariable("CLICKHOUSE_PASSWORD", "default").
		WithEnvVariable("CLICKHOUSE_DATABASE", "default").
		WithEnvVariable("JWT_SECRET_KEY", "test-secret-key-for-testing-purposes-min-32-chars").
		WithEnvVariable("JWT_ACCESS_TOKEN_TTL", "15m").
		WithEnvVariable("JWT_REFRESH_TOKEN_TTL", "168h").
		WithEnvVariable("BCRYPT_COST", "4").
		WithEnvVariable("BUNDEBUG", "0").
		WithEnvVariable("TEST_SERVER_PORT", "1324").
		WithEnvVariable("TEST_SERVER_HOST", "localhost").
		WithEnvVariable("REDIS_ADDRS", "redis:6379").
		WithEnvVariable("REDIS_PASSWORD", "").
		WithEnvVariable("NATS_CREDENTIALS_CONTENT", "").
		WithEnvVariable("NATS_STREAM_URL", "nats://localhost:4222").
		WithEnvVariable("ENCRYPTION_KEY", "test-encryption-key-must-be-32-chars-long!").
		WithEnvVariable("TELEMETRY_ENABLED", "false").
		WithEnvVariable("DATADOG_ENABLED", "false").
		WithEnvVariable("OTEL_ENABLED", "false").
		WithEnvVariable("LOG_LEVEL", "error").
		WithEnvVariable("TEST_TIMEOUT", "30s").
		WithEnvVariable("TEST_DB_TIMEOUT", "5s").
		WithEnvVariable("FEATURE_EMAIL_VERIFICATION", "false").
		WithEnvVariable("FEATURE_PASSWORD_RECOVERY", "false").
		WithEnvVariable("FEATURE_RATE_LIMITING", "false")

	// Run PostgreSQL migrations
	_, err := container.
		WithExec([]string{"goose", "-dir", "internal/storage/postgres/migrations", "postgres", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable", "up"}).
		Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to run postgres migrations: %w", err)
	}

	// Run ClickHouse migrations
	_, err = container.
		WithEnvVariable("GOOSE_DRIVER", "clickhouse").
		WithEnvVariable("GOOSE_DBSTRING", "clickhouse://default:default@clickhouse:9000").
		WithExec([]string{"goose", "-dir", "internal/storage/clickhouse/migrations", "up"}).
		Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to run clickhouse migrations: %w", err)
	}

	// Run tests with coverage
	return container.
		WithExec([]string{"go", "test", "-v", "-race", "-coverprofile=coverage.out", "-covermode=atomic", "./..."}).
		Stdout(ctx)
}

// TestWithCoverage runs tests and exports the coverage report
func (m *Zori) TestWithCoverage(ctx context.Context, source *dagger.Directory) *dagger.File {
	// Start required services
	postgres := m.postgres()
	clickhouse := m.clickhouse()
	nats := m.nats()
	redis := m.redis()

	// Build the test container with all dependencies
	container := m.testContainer(source).
		WithServiceBinding("postgres", postgres).
		WithServiceBinding("clickhouse", clickhouse).
		WithServiceBinding("nats", nats).
		WithServiceBinding("redis", redis).
		WithEnvVariable("POSTGRES_URL", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable").
		WithEnvVariable("CLICKHOUSE_URL", "clickhouse://default:default@clickhouse:9000").
		WithEnvVariable("CLICKHOUSE_USERNAME", "default").
		WithEnvVariable("CLICKHOUSE_USER", "default").
		WithEnvVariable("CLICKHOUSE_PASSWORD", "default").
		WithEnvVariable("CLICKHOUSE_DATABASE", "default").
		WithEnvVariable("JWT_SECRET_KEY", "test-secret-key-for-testing-purposes-min-32-chars").
		WithEnvVariable("JWT_ACCESS_TOKEN_TTL", "15m").
		WithEnvVariable("JWT_REFRESH_TOKEN_TTL", "168h").
		WithEnvVariable("BCRYPT_COST", "4").
		WithEnvVariable("BUNDEBUG", "0").
		WithEnvVariable("TEST_SERVER_PORT", "1324").
		WithEnvVariable("TEST_SERVER_HOST", "localhost").
		WithEnvVariable("REDIS_ADDRS", "redis:6379").
		WithEnvVariable("REDIS_PASSWORD", "").
		WithEnvVariable("NATS_CREDENTIALS_CONTENT", "").
		WithEnvVariable("NATS_STREAM_URL", "nats://nats:4222").
		WithEnvVariable("ENCRYPTION_KEY", "test-encryption-key-must-be-32-chars-long!").
		WithEnvVariable("TELEMETRY_ENABLED", "false").
		WithEnvVariable("DATADOG_ENABLED", "false").
		WithEnvVariable("OTEL_ENABLED", "false").
		WithEnvVariable("LOG_LEVEL", "error").
		WithEnvVariable("TEST_TIMEOUT", "30s").
		WithEnvVariable("TEST_DB_TIMEOUT", "5s").
		WithEnvVariable("FEATURE_EMAIL_VERIFICATION", "false").
		WithEnvVariable("FEATURE_PASSWORD_RECOVERY", "false").
		WithEnvVariable("FEATURE_RATE_LIMITING", "false")

	// Run PostgreSQL migrations
	container = container.
		WithExec([]string{"goose", "-dir", "internal/storage/postgres/migrations", "postgres", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable", "up"})

	// Run ClickHouse migrations
	container = container.
		WithEnvVariable("GOOSE_DRIVER", "clickhouse").
		WithEnvVariable("GOOSE_DBSTRING", "clickhouse://default:default@clickhouse:9000").
		WithExec([]string{"goose", "-dir", "internal/storage/clickhouse/migrations", "up"})

	// Run tests with coverage
	return container.
		WithExec([]string{"go", "test", "-v", "-race", "-coverprofile=coverage.out", "-covermode=atomic", "./..."}).
		File("coverage.out")
}

// testContainer creates a Go container with the source code and dependencies installed
func (m *Zori) testContainer(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("golang:1.24-alpine").
		WithExec([]string{"apk", "add", "--no-cache", "git"}).
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build")).
		WithExec([]string{"go", "install", "github.com/pressly/goose/v3/cmd/goose@latest"}).
		WithExec([]string{"go", "mod", "download"})
}

// postgres returns a PostgreSQL service container
func (m *Zori) postgres() *dagger.Service {
	return dag.Container().
		From("postgres:latest").
		WithEnvVariable("POSTGRES_PASSWORD", "postgres").
		WithEnvVariable("POSTGRES_DB", "postgres").
		WithExposedPort(5432).
		AsService()
}

// clickhouse returns a ClickHouse service container
func (m *Zori) clickhouse() *dagger.Service {
	return dag.Container().
		From("clickhouse/clickhouse-server:latest").
		WithEnvVariable("CLICKHOUSE_DB", "default").
		WithEnvVariable("CLICKHOUSE_USER", "default").
		WithEnvVariable("CLICKHOUSE_PASSWORD", "default").
		WithExposedPort(9000).
		AsService()
}

// nats returns a NATS JetStream service container
func (m *Zori) nats() *dagger.Service {
	return dag.Container().
		From("nats/nats:latest").
		WithExposedPort(4222).
		WithEntrypoint([]string{"nats-server", "-js", "-m", "8222", "-p", "4222"}).
		AsService()
}

// redis returns a Redis service container
func (m *Zori) redis() *dagger.Service {
	return dag.Container().
		From("redis:latest").
		WithExposedPort(6379).
		AsService()
}

// Build creates a production-ready binary
func (m *Zori) Build(ctx context.Context, source *dagger.Directory) *dagger.File {
	return dag.Container().
		From("golang:1.24-alpine").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod")).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build")).
		WithEnvVariable("CGO_ENABLED", "0").
		WithExec([]string{"go", "build", "-o", "zori", "."}).
		File("zori")
}
