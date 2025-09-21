package config

import "github.com/caarlos0/env/v11"

type Config struct {
	ClickHouseURL string `env:"CLICKHOUSE_URL"`
	PostgresURL   string `env:"POSTGRES_URL"`
}

func NewConfig() *Config {
	var cfg Config
	err := env.Parse(&cfg)

	if err != nil {
		panic(err)
	}

	return &cfg
}
