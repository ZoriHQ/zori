package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Project struct {
	bun.BaseModel `json:"-" bun:"table:projects,alias:p"`

	ID                   string     `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationID       string     `json:"organization_id" bun:",notnull" example:"660e8400-e29b-41d4-a716-446655440001"`
	Name                 string     `json:"name" bun:",notnull" example:"My Awesome Project"`
	Domain               string     `json:"domain" bun:",notnull" example:"https://example.com"`
	AllowLocalHost       bool       `json:"allow_local_host" bun:",notnull,default:false" example:"false"`
	FirstEventReceivedAt *time.Time `json:"first_event_received_at" bun:",null" example:"2024-01-15T10:30:00Z"`

	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}
