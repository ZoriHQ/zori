package models

import (
	"github.com/uptrace/bun"
)

type Account struct {
	bun.BaseModel `json:"-" bun:"table:accounts,alias:a"`

	ID            string `json:"id" bun:",pk"`
	Email         string `json:"email"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	EmailVerified bool   `json:"email_verified"`
}

func (a *Account) FullName() string {
	if a.FirstName != "" && a.LastName != "" {
		return a.FirstName + " " + a.LastName
	}
	if a.FirstName != "" {
		return a.FirstName
	}
	if a.LastName != "" {
		return a.LastName
	}
	return a.Email
}

func (a *Account) IsEmailVerified() bool {
	return a.EmailVerified
}

func (*Account) TableName() string {
	return "accounts"
}
