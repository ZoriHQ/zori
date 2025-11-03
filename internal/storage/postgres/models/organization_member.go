package models

import (
	"time"

	"github.com/uptrace/bun"
)

type OrganizationMember struct {
	bun.BaseModel `json:"-" bun:"table:organization_members,alias:om"`

	ID             string    `json:"id" bun:",pk,type:uuid,default:gen_random_uuid()"`
	OrganizationID string    `json:"organization_id" bun:",notnull" validate:"required"`
	AccountID      string    `json:"account_id" bun:",notnull" validate:"required"`
	Role           string    `json:"role" bun:",notnull,default:'member'" validate:"required,oneof=owner admin member"`
	JoinedAt       time.Time `json:"joined_at" bun:",nullzero,notnull,default:current_timestamp"`

	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Account      *Account      `json:"account,omitempty" bun:"rel:belongs-to,join:account_id=id"`
}

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

func (om *OrganizationMember) IsOwner() bool {
	return om.Role == RoleOwner
}

func (om *OrganizationMember) IsAdmin() bool {
	return om.Role == RoleAdmin
}

func (om *OrganizationMember) IsMember() bool {
	return om.Role == RoleMember
}

func (om *OrganizationMember) HasAdminRights() bool {
	return om.Role == RoleOwner || om.Role == RoleAdmin
}

func (om *OrganizationMember) IsValidRole() bool {
	return om.Role == RoleOwner || om.Role == RoleAdmin || om.Role == RoleMember
}

func (*OrganizationMember) TableName() string {
	return "organization_members"
}
