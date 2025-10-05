package models

import (
	"time"

	"github.com/uptrace/bun"
)

type AccessToken struct {
	bun.BaseModel `json:"-" bun:"table:access_tokens,alias:at"`

	ID             string     `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()"`
	OrganizationID string     `json:"organization_id" bun:",notnull" validate:"required"`
	Name           string     `json:"first_name" bun:",notnull"`
	Secret         string     `json:"secret" bun:","`
	CreatedAt      time.Time  `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	LastUsed       *time.Time `json:"last_used" bun:",nullzero,default:null"`

	Organizations Organization `json:"organizations" bun:"o2m:organization_members,join:AccessToken=Organization"`
}
