package types

import (
	"time"
	"zori/internal/storage/postgres/models"
)

type PaymentProviderResponse struct {
	ID             string              `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProjectID      string              `json:"project_id" example:"660e8400-e29b-41d4-a716-446655440001"`
	OrganizationID string              `json:"organization_id" example:"770e8400-e29b-41d4-a716-446655440002"`
	ProviderType   models.ProviderType `json:"provider_type" example:"stripe"`
	IsActive       bool                `json:"is_active" example:"true"`
	LastSyncedAt   *time.Time          `json:"last_synced_at" example:"2024-01-15T10:30:00Z"`
	CreatedAt      time.Time           `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt      time.Time           `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

type ListPaymentProvidersResponse struct {
	Providers []*PaymentProviderResponse `json:"providers"`
	Total     int                        `json:"total" example:"3"`
}

type SyncStatusResponse struct {
	ProviderID     string    `json:"provider_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status         string    `json:"status" example:"completed"`
	PaymentsSynced int       `json:"payments_synced" example:"150"`
	LastSyncedAt   time.Time `json:"last_synced_at" example:"2024-01-15T10:30:00Z"`
	Message        string    `json:"message,omitempty" example:"Successfully synced 150 payments"`
}

type ProviderField struct {
	Name        string `json:"name" example:"api_key"`
	Label       string `json:"label" example:"API Key"`
	Type        string `json:"type" example:"password"`
	Required    bool   `json:"required" example:"true"`
	Placeholder string `json:"placeholder,omitempty" example:"sk_live_xxxxx"`
	HelpText    string `json:"help_text,omitempty" example:"Your Stripe secret key"`
}

type ProviderInstructionsResponse struct {
	ProviderType     models.ProviderType `json:"provider_type" example:"stripe"`
	ConnectionMethod string              `json:"connection_method" example:"oauth"`
	OAuthURL         string              `json:"oauth_url,omitempty" example:"https://connect.stripe.com/oauth/authorize?..."`
	Fields           []ProviderField     `json:"fields,omitempty"`
	Instructions     string              `json:"instructions,omitempty" example:"Click the button below to connect your Stripe account"`
}

type StripeConnectCallbackResponse struct {
	Success        bool                     `json:"success" example:"true"`
	Message        string                   `json:"message" example:"Successfully connected Stripe account"`
	Provider       *PaymentProviderResponse `json:"provider,omitempty"`
	BackfillStatus string                   `json:"backfill_status,omitempty" example:"started"`
}

func ToPaymentProviderResponse(provider *models.PaymentProvider) *PaymentProviderResponse {
	return &PaymentProviderResponse{
		ID:             provider.ID,
		ProjectID:      provider.ProjectID,
		OrganizationID: provider.OrganizationID,
		ProviderType:   provider.ProviderType,
		IsActive:       provider.IsActive,
		LastSyncedAt:   provider.LastSyncedAt,
		CreatedAt:      provider.CreatedAt,
		UpdatedAt:      provider.UpdatedAt,
	}
}
