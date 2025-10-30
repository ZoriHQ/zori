package data

import (
	"context"
	"database/sql"

	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"

	"github.com/uptrace/bun"
)

type AuthData struct {
	db *bun.DB
}

func NewAuthData(db *postgres.PostgresDB) *AuthData {
	return &AuthData{db: db.DB}
}

func (a *AuthData) GetSystemConfig(ctx context.Context) (*models.System, error) {
	system := &models.System{}
	err := a.db.NewSelect().
		Model(system).
		Limit(1).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return system, nil
}
