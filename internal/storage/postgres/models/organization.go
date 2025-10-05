package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Organization struct {
	bun.BaseModel `json:"-" bun:"table:organizations,alias:o"`

	ID        string    `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()"`
	Name      string    `json:"name" bun:",notnull" validate:"required,min=1,max=255"`
	Slug      string    `json:"slug" bun:",unique,notnull" validate:"required,min=1,max=255,alphanum"`
	CreatedAt time.Time `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `json:"updated_at" bun:",nullzero,notnull,default:current_timestamp"`

	// Relations
	Members []Account `json:"members,omitempty" bun:"m2m:organization_members,join:Organization=Account"`
}

// IsValidSlug checks if the slug is valid
func (o *Organization) IsValidSlug() bool {
	return len(o.Slug) > 0 && len(o.Slug) <= 255
}

// TableName returns the table name for the Organization model
func (*Organization) TableName() string {
	return "organizations"
}
