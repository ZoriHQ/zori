package data

import (
	"marker/internal/ctx"
	"marker/internal/storage/postgres"
	"marker/internal/storage/postgres/models"

	"github.com/uptrace/bun"
)

type OrganizationData struct {
	db *bun.DB
}

func NewOrganizationData(db *postgres.PostgresDB) *OrganizationData {
	return &OrganizationData{
		db: db.DB,
	}
}

func (o *OrganizationData) GetOrganizationByID(c *ctx.Ctx, id string) (*models.Organization, error) {
	var model models.Organization
	err := o.db.NewSelect().Model(&model).Where("id = ?", id).Scan(c, &model)
	return &model, err
}
