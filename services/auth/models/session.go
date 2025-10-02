package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Session struct {
	bun.BaseModel `bun:"table:sessions,alias:s"`

	ID        string    `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()"`
	AccountID string    `json:"account_id" bun:",notnull" validate:"required"`
	ExpiresAt time.Time `json:"expires_at" bun:",notnull" validate:"required"`
	CreatedAt time.Time `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `json:"updated_at" bun:",nullzero,notnull,default:current_timestamp"`

	// Relations
	Account *Account `json:"account,omitempty" bun:"rel:belongs-to,join:account_id=id"`
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the session is valid (not expired)
func (s *Session) IsValid() bool {
	return !s.IsExpired()
}

// TimeUntilExpiry returns the duration until the session expires
func (s *Session) TimeUntilExpiry() time.Duration {
	if s.IsExpired() {
		return 0
	}
	return time.Until(s.ExpiresAt)
}

// TableName returns the table name for the Session model
func (*Session) TableName() string {
	return "sessions"
}
