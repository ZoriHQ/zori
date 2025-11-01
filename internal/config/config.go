package config

import (
	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
)

type ContextKey string

const ResetAuthKey ContextKey = "reset-auth"

type Config struct {
	ClickHouseURL      string `env:"CLICKHOUSE_URL,required"`
	ClickHouseUsername string `env:"CLICKHOUSE_USERNAME,required"`
	ClickHousePassword string `env:"CLICKHOUSE_PASSWORD,required"`
	ClickHouseDatabase string `env:"CLICKHOUSE_DATABASE" envDefault:"default"`
	PostgresURL        string `env:"POSTGRES_URL,required"`

	RedisADDS string `env:"REDIS_ADDRS,required"`
	RedisPASS string `env:"REDIS_PASSWORD,required"`

	// Clerk Auth Configuration (not required in OSS mode)
	ClerkSecretKey string `env:"CLERK_SECRET_KEY"`

	// OSS Auth Configuration (only used when ZoriOSS is true)
	JWTSecret string `env:"JWT_SECRET"`

	NatsCredentialsContent string `env:"NATS_CREDENTIALS_CONTENT,required"`
	NatsStreamURL          string `env:"NATS_STREAM_URL,required"`

	// Encryption Configuration (for payment provider credentials)
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`

	ZoriOSS     bool   `env:"ZORI_IS_OSS"`
	ZoriAPIHost string `env:"ZORI_API_HOST,required"`

	ZoriStripeConnect              bool   `env:"ZORI_STRIPE_CONNECT"`
	ZoriStripeConnectSecretKey     string `env:"ZORI_STRIPE_CONNECT_SECRET_KEY"`
	ZoriStripeConnectClientID      string `env:"ZORI_STRIPE_CONNECT_CLIENT_ID"`
	ZoriStripeConnectWebhookSecret string `env:"ZORI_STRIPE_CONNECT_WEBHOOK_SECRET"`

	// Metrics Configuration (for Grafana Cloud)
	MetricsEnabled         bool   `env:"METRICS_ENABLED" envDefault:"false"`
	MetricsPort            string `env:"METRICS_PORT" envDefault:"9090"`
	GrafanaCloudRemoteURL  string `env:"GRAFANA_CLOUD_REMOTE_WRITE_URL"`
	GrafanaCloudUsername   string `env:"GRAFANA_CLOUD_USERNAME"`
	GrafanaCloudAPIKey     string `env:"GRAFANA_CLOUD_API_KEY"`
}

func NewConfig() *Config {
	var cfg Config
	err := env.Parse(&cfg)

	if err != nil {
		panic(err)
	}

	return &cfg
}
