package models

import (
	"time"

	"github.com/uptrace/bun"
)

// System represents OSS deployment system configuration
type System struct {
	bun.BaseModel `json:"-" bun:"table:system,alias:s"`

	ID                string    `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()"`
	ProjectsLimit     *int      `json:"projects_limit,omitempty" bun:",nullzero"`
	AdminUsername     *string   `json:"admin_username,omitempty" bun:",nullzero"`
	AdminPasswordHash *string   `json:"-" bun:",nullzero"`
	DefaultOrgID      *string   `json:"default_org_id,omitempty" bun:",type:varchar(255),nullzero"`
	CreatedAt         time.Time `json:"created_at" bun:",notnull,default:current_timestamp"`
	UpdatedAt         time.Time `json:"updated_at" bun:",notnull,default:current_timestamp"`
}

func (*System) TableName() string {
	return "system"
}
