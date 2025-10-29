package config

import (
	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	ClickHouseURL      string `env:"CLICKHOUSE_URL,required"`
	ClickHouseUsername string `env:"CLICKHOUSE_USERNAME,required"`
	ClickHousePassword string `env:"CLICKHOUSE_PASSWORD,required"`
	ClickHouseDatabase string `env:"CLICKHOUSE_DATABASE" envDefault:"default"`
	PostgresURL        string `env:"POSTGRES_URL,required"`

	RedisADDS string `env:"REDIS_ADDRS,required"`
	RedisPASS string `env:"REDIS_PASSWORD,required"`

	// Stack Auth Configuration
	StackAuthProjectID string `env:"STACK_AUTH_PROJECT_ID,required"`
	StackAuthJWKSURL   string `env:"STACK_AUTH_JWKS_URL"`

	NatsCredentialsContent string `env:"NATS_CREDENTIALS_CONTENT,required"`
	NatsStreamURL          string `env:"NATS_STREAM_URL,required"`

	// Encryption Configuration (for payment provider credentials)
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`

	ZoriOSS bool `env:"ZORI_IS_OSS"`

	ZoriStripeConnect                     bool   `env:"ZORI_STRIPE_CONNECT"`
	ZoriStripeConnectSecretKey            string `env:"ZORI_STRIPE_CONNECT_SECRET_KEY"`
	ZoriStripeConnectSandboxKey           string `env:"ZORI_STRIPE_CONNECT_SANDBOX_SECRET_KEY"`
	ZoriStripeConnectClientID             string `env:"ZORI_STRIPE_CONNECT_CLIENT_ID"`
	ZoriStripeConnectWebhookSecret        string `env:"ZORI_STRIPE_CONNECT_WEBHOOK_SECRET"`
	ZoriStripeConnectWebhookSandboxSecret string `env:"ZORI_STRIPE_CONNECT_WEBHOOK_SANDBOX_SECRET"`
	ZoriStripeConnectClientSandboxID      string `env:"ZORI_STRIPE_CONNECT_CLIENT_SANDBOX_ID"`
}

func (c *Config) GetStackAuthJWKSURL() string {
	if c.StackAuthJWKSURL != "" {
		return c.StackAuthJWKSURL
	}
	// Default JWKS URL based on project ID
	return "https://api.stack-auth.com/api/v1/projects/" + c.StackAuthProjectID + "/.well-known/jwks.json"
}

func NewConfig() *Config {
	var cfg Config
	err := env.Parse(&cfg)

	if err != nil {
		panic(err)
	}

	return &cfg
}
