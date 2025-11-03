package types

import "zori/internal/storage/postgres/models"

type CreatePaymentProviderRequest struct {
	ProjectID     string              `json:"project_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProviderType  models.ProviderType `json:"provider_type" validate:"required,oneof=stripe paddle paypal lemon_squeezy square" example:"stripe"`
	APIKey        string              `json:"api_key" validate:"required,min=10" example:"sk_test_xxxxx"`
	WebhookSecret string              `json:"webhook_secret" validate:"required,min=10" example:"whsec_xxxxx"`
}

type UpdatePaymentProviderRequest struct {
	APIKey        *string `json:"api_key,omitempty" validate:"omitempty,min=10" example:"sk_test_xxxxx"`
	WebhookSecret *string `json:"webhook_secret,omitempty" validate:"omitempty,min=10" example:"whsec_xxxxx"`
	IsActive      *bool   `json:"is_active,omitempty" example:"true"`
}

type SyncPaymentProviderRequest struct {
	FullSync bool `json:"full_sync" example:"false"`
}

type GetProviderInstructionsRequest struct {
	ProviderType string `query:"provider_type" validate:"required,oneof=stripe paddle paypal lemon_squeezy square" example:"stripe"`
}

type StripeConnectCallbackRequest struct {
	Code  string `query:"code" validate:"required" example:"ac_xxxxx"`
	State string `query:"state" validate:"required" example:"random_state_string"`
}
