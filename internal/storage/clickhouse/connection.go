package clickhouse

import (
	"context"
	"crypto/tls"

	"zori/internal/config"

	goclick "github.com/ClickHouse/clickhouse-go/v2"
)

type ClickhouseDB struct {
	conn goclick.Conn
}

func NewClickhouseDB(cfg *config.Config) *ClickhouseDB {
	if cfg.ClickHouseURL == "" {
		panic("CLICKHOUSE_URL is required")
	}

	clickDbConn, err := goclick.Open(&goclick.Options{
		Addr: []string{cfg.ClickHouseURL},
		Auth: goclick.Auth{
			Username: "default",
			Password: cfg.ClickHousePassword,
		},
		Protocol: goclick.Native,
		TLS:      &tls.Config{},
		// TODO:: move to configs
		Debug: true,
	})

	if err != nil {
		panic(err)
	}

	if err = clickDbConn.Ping(context.Background()); err != nil {
		panic(err)
	}

	return &ClickhouseDB{conn: clickDbConn}
}

func (p *ClickhouseDB) Db() goclick.Conn {
	return p.conn
}

func (p *ClickhouseDB) Close() error {
	return p.conn.Close()
}

func (p *ClickhouseDB) Ping(ctx context.Context) error {
	return p.conn.Ping(ctx)
}
