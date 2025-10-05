package data

import (
	"context"
	"marker/internal/ctx"
	"marker/internal/storage/postgres"
	"marker/internal/storage/postgres/models"
	"marker/services/projects/types"
	"time"

	"github.com/uptrace/bun"
)

type ProjectData struct {
	db *bun.DB
}

func NewProjectData(db *postgres.PostgresDB) *ProjectData {
	return &ProjectData{db: db.DB}
}

func (p *ProjectData) ListOrganizationProjects(orgID string) ([]*ProjectData, error) {
	var projects []*ProjectData
	err := p.db.NewSelect().Model(&projects).Where("organization_id = ?", orgID).Scan(context.Background())
	return projects, err
}

func (p *ProjectData) CreateProject(c *ctx.Ctx, req *types.CreateProjectRequest) error {
	_, err := p.db.NewInsert().Model(&models.Project{
		Name:           req.Name,
		Domain:         req.WebsiteURL,
		OrganizationID: c.OrgID(),
	}).Exec(c)
	return err
}

func (p *ProjectData) DeleteProject(c *ctx.Ctx, projectID string) error {
	_, err := p.db.NewDelete().Model(&models.Project{
		ID: projectID,
	}).Where("organization_id = ?", c.OrgID()).Exec(c)
	return err
}

func (p *ProjectData) SetFirstEventReceived(c *ctx.Ctx, projectID string) error {
	_, err := p.db.NewUpdate().Model(&models.Project{
		ID: projectID,
	}).Set("first_event_received_at = ?", time.Now()).Exec(c)
	return err
}
