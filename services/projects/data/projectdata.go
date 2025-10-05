package data

import (
	"context"
	"time"
	"zori/internal/ctx"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"
	"zori/services/projects/types"

	"github.com/uptrace/bun"
)

type ProjectData struct {
	db *bun.DB
}

func NewProjectData(db *postgres.PostgresDB) *ProjectData {
	return &ProjectData{db: db.DB}
}

func (p *ProjectData) ListOrganizationProjects(orgID string) ([]*models.Project, error) {
	var projects []*models.Project
	err := p.db.NewSelect().
		Model(&projects).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(context.Background())
	return projects, err
}

func (p *ProjectData) GetProject(ctx context.Context, projectID string, orgID string) (*models.Project, error) {
	project := &models.Project{}
	err := p.db.NewSelect().
		Model(project).
		Where("id = ?", projectID).
		Where("organization_id = ?", orgID).
		Scan(ctx)
	return project, err
}

func (p *ProjectData) CreateProject(c *ctx.Ctx, req *types.CreateProjectRequest) (*models.Project, error) {
	project := &models.Project{
		Name:           req.Name,
		Domain:         req.WebsiteURL,
		OrganizationID: c.OrgID(),
		AllowLocalHost: req.AllowLocalHost,
	}

	_, err := p.db.NewInsert().
		Model(project).
		Returning("*").
		Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return project, nil
}

func (p *ProjectData) UpdateProject(ctx context.Context, projectID string, orgID string, req *types.UpdateProjectRequest) (*models.Project, error) {
	project := &models.Project{}

	query := p.db.NewUpdate().
		Model(project).
		Where("id = ?", projectID).
		Where("organization_id = ?", orgID).
		Returning("*")

	if req.Name != "" {
		query = query.Set("name = ?", req.Name)
	}
	if req.WebsiteURL != "" {
		query = query.Set("domain = ?", req.WebsiteURL)
	}
	query = query.Set("allow_local_host = ?", req.AllowLocalHost)

	_, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (p *ProjectData) DeleteProject(c *ctx.Ctx, projectID string) error {
	_, err := p.db.NewDelete().
		Model(&models.Project{}).
		Where("id = ?", projectID).
		Where("organization_id = ?", c.OrgID()).
		Exec(context.Background())
	return err
}

func (p *ProjectData) SetFirstEventReceived(c *ctx.Ctx, projectID string) error {
	_, err := p.db.NewUpdate().
		Model(&models.Project{}).
		Set("first_event_received_at = ?", time.Now()).
		Where("id = ?", projectID).
		Where("organization_id = ?", c.OrgID()).
		Exec(context.Background())
	return err
}

func (p *ProjectData) ProjectExists(ctx context.Context, projectID string, orgID string) (bool, error) {
	return p.db.NewSelect().
		Model(&models.Project{}).
		Where("id = ?", projectID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
}

func (p *ProjectData) CountOrganizationProjects(ctx context.Context, orgID string) (int, error) {

	return p.db.NewSelect().
		Model(&models.Project{}).
		Where("organization_id = ?", orgID).
		Count(ctx)
}
