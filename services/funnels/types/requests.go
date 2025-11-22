package types

import "zori/internal/storage/postgres/models"

type CreateFunnelRequest struct {
	ProjectID     string            `json:"project_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name          string            `json:"name" validate:"required,min=1,max=255" example:"Checkout Funnel"`
	Description   *string           `json:"description,omitempty" example:"Tracks user checkout flow"`
	FunnelType    models.FunnelType `json:"funnel_type" validate:"required,oneof=sequential depth" example:"sequential"`
	WindowSeconds *int              `json:"window_seconds,omitempty" validate:"omitempty,min=60,max=2592000" example:"86400"`
	IsStrict      *bool             `json:"is_strict,omitempty" example:"false"`
	Steps         []StepRequest     `json:"steps,omitempty" validate:"required_if=FunnelType sequential,dive"`
	DepthConfig   *DepthConfig      `json:"depth_config,omitempty" validate:"required_if=FunnelType depth"`
}

type StepRequest struct {
	Name      string        `json:"name" validate:"required,min=1,max=255" example:"View Product"`
	Condition StepCondition `json:"condition" validate:"required"`
}

type UpdateFunnelRequest struct {
	Name          *string           `json:"name,omitempty" validate:"omitempty,min=1,max=255" example:"Checkout Funnel"`
	Description   *string           `json:"description,omitempty" example:"Tracks user checkout flow"`
	FunnelType    *models.FunnelType `json:"funnel_type,omitempty" validate:"omitempty,oneof=sequential depth" example:"sequential"`
	WindowSeconds *int              `json:"window_seconds,omitempty" validate:"omitempty,min=60,max=2592000" example:"86400"`
	IsStrict      *bool             `json:"is_strict,omitempty" example:"false"`
	Steps         []StepRequest     `json:"steps,omitempty" validate:"omitempty,dive"`
	DepthConfig   *DepthConfig      `json:"depth_config,omitempty"`
}

type AnalyzeFunnelRequest struct {
	ProjectID string `json:"project_id" query:"project_id" form:"project_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	TimeRange string `json:"time_range" query:"time_range" form:"time_range" validate:"required,oneof=last_hour today yesterday last_7_days last_30_days last_90_days" example:"last_30_days"`
}

type ListFunnelsRequest struct {
	ProjectID string `json:"project_id" query:"project_id" form:"project_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}
