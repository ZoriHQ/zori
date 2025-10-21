package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// JSONB represents a PostgreSQL JSONB column
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// Visitor represents an identified visitor in PostgreSQL
type Visitor struct {
	bun.BaseModel `bun:"table:visitors,alias:v"`

	// Primary identity
	VisitorID string `bun:"visitor_id,pk"`

	// Organization hierarchy
	ProjectID      string `bun:"project_id,notnull"`
	OrganizationID string `bun:"organization_id,notnull"`

	// User identity fields (nullable - only set when visitor is identified)
	UserID     *string `bun:"user_id"`
	ExternalID *string `bun:"external_id"`
	Email      *string `bun:"email"`
	EmailHash  *string `bun:"email_hash"` // SHA256 hash for privacy-safe matching

	// Optional profile fields
	Name  *string `bun:"name"`
	Phone *string `bun:"phone"`

	// Metadata stored as JSONB for flexibility
	CustomTraits JSONB `bun:"custom_traits,type:jsonb,default:'{}'"`

	// Timestamps
	FirstIdentifiedAt *time.Time `bun:"first_identified_at"`
	LastIdentifiedAt  *time.Time `bun:"last_identified_at"`
	CreatedAt         time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt         time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
