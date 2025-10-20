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
