package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Project struct {
	bun.BaseModel `bun:"table:accounts,alias:a"`

	ID                   string     `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()"`
	OrganizationID       string     `json:"organization_id" bun:",notnull"`
	Name                 string     `json:"name" bun:",notnull"`
	Domain               string     `json:"domain" bun:",notnull"`
	AllowLocalHost       bool       `json:"allow_local_host" bun:",notnull,default:false"`
	FirstEventReceivedAt *time.Time `json:"first_event_received_at" bun:",null"`

	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}
