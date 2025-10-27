package models

import (
	"github.com/uptrace/bun"
)

// Account represents user account data from Stack Auth (external auth provider).
// This model is used for in-memory representation only - accounts are NOT stored in our database.
// All account data comes from Stack Auth JWT tokens and is managed externally.
// Note: Bun tags are kept for backward compatibility but the table no longer exists.
type Account struct {
	bun.BaseModel `json:"-" bun:"table:accounts,alias:a"`

	ID            string `json:"id" bun:",pk"`
	Email         string `json:"email"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	EmailVerified bool   `json:"email_verified"`
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
