package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type FunnelType string

const (
	FunnelTypeSequential FunnelType = "sequential"
	FunnelTypeDepth      FunnelType = "depth"
)

type Funnel struct {
	bun.BaseModel `json:"-" bun:"table:funnels,alias:f"`

	ID             string          `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProjectID      string          `json:"project_id" bun:",notnull,type:uuid" example:"660e8400-e29b-41d4-a716-446655440001"`
	OrganizationID string          `json:"organization_id" bun:",notnull" example:"770e8400-e29b-41d4-a716-446655440002"`
	Name           string          `json:"name" bun:",notnull" example:"Checkout Funnel"`
	Description    *string         `json:"description" example:"Tracks user checkout flow"`
	FunnelType     FunnelType      `json:"funnel_type" bun:",notnull,default:'sequential'" example:"sequential"`
	WindowSeconds  int             `json:"window_seconds" bun:",notnull,default:86400" example:"86400"`
	IsStrict       bool            `json:"is_strict" bun:",notnull,default:false" example:"false"`
	DepthConfig    json.RawMessage `json:"depth_config,omitempty" bun:"type:jsonb"`
	CreatedAt      time.Time       `json:"created_at" bun:",notnull,default:current_timestamp" example:"2024-01-15T10:30:00Z"`
	UpdatedAt      time.Time       `json:"updated_at" bun:",notnull,default:current_timestamp" example:"2024-01-15T10:30:00Z"`

	Steps []*FunnelStep `json:"steps,omitempty" bun:"rel:has-many,join:id=funnel_id"`
}

type FunnelStep struct {
	bun.BaseModel `json:"-" bun:"table:funnel_steps,alias:fs"`

	ID        string          `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	FunnelID  string          `json:"funnel_id" bun:",notnull,type:uuid" example:"660e8400-e29b-41d4-a716-446655440001"`
	StepOrder int             `json:"step_order" bun:",notnull" example:"1"`
	Name      string          `json:"name" bun:",notnull" example:"View Product"`
	Condition json.RawMessage `json:"condition" bun:",notnull,type:jsonb"`
	CreatedAt time.Time       `json:"created_at" bun:",notnull,default:current_timestamp" example:"2024-01-15T10:30:00Z"`
}
