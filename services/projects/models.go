package projects

import "github.com/uptrace/bun"

type Project struct {
	bun.BaseModel `bun:"table:projects,alias:p"`

	ID          string `json:"id" bun:",pk,autoincrement"`
	Name        string `json:"name" bun:",notnull"`
	OwnerID     string `json:"owner_id" bun:",notnull"`
	Description string `json:"description"`

	Website string `json:"website" bun:",notnull"`
}
