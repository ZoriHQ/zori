package data

import (
	"zori/internal/ctx"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"

	"github.com/uptrace/bun"
)

type AccountData struct {
	db *bun.DB
}

func NewAccountData(db *postgres.PostgresDB) *AccountData {
	return &AccountData{
		db: db.DB,
	}
}

func (o *AccountData) GetAccountByID(c *ctx.Ctx, id string) (*models.Account, error) {
	var model models.Account
	err := o.db.NewSelect().Model(&model).Where("id = ?", id).Scan(c, &model)
	return &model, err
}
