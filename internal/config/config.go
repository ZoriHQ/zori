package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	ClickHouseURL string `env:"CLICKHOUSE_URL,required"`
	PostgresURL   string `env:"POSTGRES_URL,required"`

	// JWT Configuration
	JWTSecretKey       string        `env:"JWT_SECRET_KEY" envDefault:"your-super-secret-key-change-in-production-min-32-chars"`
	JWTAccessTokenTTL  time.Duration `env:"JWT_ACCESS_TOKEN_TTL" envDefault:"15m"`
	JWTRefreshTokenTTL time.Duration `env:"JWT_REFRESH_TOKEN_TTL" envDefault:"168h"`

	NatsCredentialsContent string `env:"NATS_CREDENTIALS_CONTENT,required"`
	NatsStreamURL          string `env:"NATS_STREAM_URL,required"`

	// Bcrypt Configuration
	BcryptCost int `env:"BCRYPT_COST" envDefault:"12"`
}

func NewConfig() *Config {
	var cfg Config
	err := env.Parse(&cfg)

	if err != nil {
		panic(err)
	}

	return &cfg
}
