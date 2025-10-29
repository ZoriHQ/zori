package models

import (
	"time"

	"github.com/uptrace/bun"
)

// ProviderType represents the type of payment provider
type ProviderType string

const (
	ProviderTypeStripe       ProviderType = "stripe"
	ProviderTypePaddle       ProviderType = "paddle"
	ProviderTypePayPal       ProviderType = "paypal"
	ProviderTypeLemonSqueezy ProviderType = "lemon_squeezy"
	ProviderTypeSquare       ProviderType = "square"
)

// PaymentProvider represents a payment provider integration for a project
type PaymentProvider struct {
	bun.BaseModel `json:"-" bun:"table:payment_providers,alias:pp"`

	ID string `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	// AccountID represents external account id of the linked payment provider
	AccountID      string       `json:"account_id" bun:",notnull" example:"880e8400-e29b-41d4-a716-446655440003"`
	ProjectID      string       `json:"project_id" bun:",notnull" example:"660e8400-e29b-41d4-a716-446655440001"`
	OrganizationID string       `json:"organization_id" bun:",notnull" example:"770e8400-e29b-41d4-a716-446655440002"`
	ProviderType   ProviderType `json:"provider_type" bun:",notnull" example:"stripe"`

	// Encrypted credentials - these should never be returned in API responses
	APIKeyEncrypted        string `json:"-" bun:",notnull"`
	WebhookSecretEncrypted string `json:"-" bun:",notnull"`

	// Status fields
	IsActive     bool       `json:"is_active" bun:",notnull,default:true" example:"true"`
	LastSyncedAt *time.Time `json:"last_synced_at" bun:",null" example:"2024-01-15T10:30:00Z"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" bun:",notnull,default:current_timestamp" example:"2024-01-15T10:30:00Z"`
	UpdatedAt time.Time `json:"updated_at" bun:",notnull,default:current_timestamp" example:"2024-01-15T10:30:00Z"`

	// Relations
	Project *Project `json:"project,omitempty" bun:"rel:belongs-to,join:project_id=id"`
}
