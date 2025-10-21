package models

import (
	"time"

	"github.com/uptrace/bun"
)

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
	CustomTraits map[string]interface{} `bun:"custom_traits,type:jsonb,default:'{}'"`

	// Timestamps
	FirstIdentifiedAt *time.Time `bun:"first_identified_at"`
	LastIdentifiedAt  *time.Time `bun:"last_identified_at"`
	CreatedAt         time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt         time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
