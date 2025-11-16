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

	ZoriStripeApp              bool   `env:"ZORI_STRIPE_APP"`
	ZoriStripeAppSecretKey     string `env:"ZORI_STRIPE_APP_SECRET_KEY"`
	ZoriStripeAppInstallLink   string `env:"ZORI_STRIPE_APP_INSTALL_LINK"`
	ZoriStripeAppWebhookSecret string `env:"ZORI_STRIPE_APP_WEBHOOK_SECRET"`

	// Metrics Configuration (for Grafana Cloud)
	MetricsEnabled         bool   `env:"METRICS_ENABLED" envDefault:"false"`
	MetricsPort            string `env:"METRICS_PORT" envDefault:"9090"`
	GrafanaCloudRemoteURL  string `env:"GRAFANA_CLOUD_REMOTE_WRITE_URL"`
	GrafanaCloudUsername   string `env:"GRAFANA_CLOUD_USERNAME"`
	GrafanaCloudAPIKey     string `env:"GRAFANA_CLOUD_API_KEY"`

	// OpenTelemetry Configuration (for Grafana Tempo)
	OTelEnabled            bool    `env:"OTEL_ENABLED" envDefault:"true"`
	OTelEndpoint           string  `env:"OTEL_ENDPOINT" envDefault:"localhost:4317"`
	OTelServiceName        string  `env:"OTEL_SERVICE_NAME" envDefault:"zori"`
	OTelServiceVersion     string  `env:"OTEL_SERVICE_VERSION" envDefault:"0.2.4"`
	OTelEnvironment        string  `env:"OTEL_ENVIRONMENT" envDefault:"development"`
	OTelHTTPSamplingRate   float64 `env:"OTEL_HTTP_SAMPLING_RATE" envDefault:"1.0"`
	OTelIngestSamplingRate float64 `env:"OTEL_INGEST_SAMPLING_RATE" envDefault:"0.1"`

	// Logging Configuration (for Grafana Loki)
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"json"`
	LokiURL   string `env:"LOKI_URL"`
}

func NewConfig() *Config {
	var cfg Config
	err := env.Parse(&cfg)

	if err != nil {
		panic(err)
	}

	return &cfg
}
