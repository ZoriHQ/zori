package services

import (
	"zori/internal/ctx"
	"zori/internal/storage/postgres/models"
	"zori/services/organizations/data"
)

type AccountService struct {
	data *data.AccountData
}

func NewAccountService(data *data.AccountData) *AccountService {
	return &AccountService{
		data: data,
	}
}

func (s *AccountService) GetAccountByID(c *ctx.Ctx, id string) (*models.Account, error) {
	return s.data.GetAccountByID(c, id)
}
