package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Account struct {
	bun.BaseModel `json:"-" bun:"table:accounts,alias:a"`

	ID            string    `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()"`
	Email         string    `json:"email" bun:",unique,notnull" validate:"required,email"`
	PasswordHash  string    `json:"-" bun:",notnull"`
	FirstName     string    `json:"first_name" bun:",nullzero"`
	LastName      string    `json:"last_name" bun:",nullzero"`
	EmailVerified bool      `json:"email_verified" bun:",default:false"`
	CreatedAt     time.Time `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `json:"updated_at" bun:",nullzero,notnull,default:current_timestamp"`

	// Relations
	Organizations []Organization `json:"organizations,omitempty" bun:"m2m:organization_members,join:Account=Organization"`
	Sessions      []Session      `json:"-" bun:"rel:has-many,join:id=account_id"`
}

// FullName returns the account's full name
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

// IsEmailVerified returns whether the account's email is verified
func (a *Account) IsEmailVerified() bool {
	return a.EmailVerified
}

// TableName returns the table name for the Account model
func (*Account) TableName() string {
	return "accounts"
}
