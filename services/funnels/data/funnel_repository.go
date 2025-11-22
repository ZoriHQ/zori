package data

import (
	"context"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"

	"github.com/uptrace/bun"
)

type FunnelRepository struct {
	db *bun.DB
}

func NewFunnelRepository(db *postgres.PostgresDB) *FunnelRepository {
	return &FunnelRepository{db: db.DB}
}

func (r *FunnelRepository) CreateFunnel(ctx context.Context, funnel *models.Funnel) error {
	_, err := r.db.NewInsert().
		Model(funnel).
		Returning("*").
		Exec(ctx)
	return err
}

func (r *FunnelRepository) CreateFunnelSteps(ctx context.Context, steps []*models.FunnelStep) error {
	if len(steps) == 0 {
		return nil
	}
	_, err := r.db.NewInsert().
		Model(&steps).
		Exec(ctx)
	return err
}

func (r *FunnelRepository) GetFunnel(ctx context.Context, funnelID string, orgID string) (*models.Funnel, error) {
	funnel := &models.Funnel{}
	err := r.db.NewSelect().
		Model(funnel).
		Relation("Steps", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("step_order ASC")
		}).
		Where("f.id = ?", funnelID).
		Where("f.organization_id = ?", orgID).
		Scan(ctx)
	return funnel, err
}

func (r *FunnelRepository) GetFunnelByProjectAndName(ctx context.Context, projectID string, name string) (*models.Funnel, error) {
	funnel := &models.Funnel{}
	err := r.db.NewSelect().
		Model(funnel).
		Where("project_id = ?", projectID).
		Where("name = ?", name).
		Scan(ctx)
	return funnel, err
}

func (r *FunnelRepository) ListFunnelsByProject(ctx context.Context, projectID string, orgID string) ([]*models.Funnel, error) {
	var funnels []*models.Funnel
	err := r.db.NewSelect().
		Model(&funnels).
		Relation("Steps", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("step_order ASC")
		}).
		Where("f.project_id = ?", projectID).
		Where("f.organization_id = ?", orgID).
		Order("f.created_at DESC").
		Scan(ctx)
	return funnels, err
}

func (r *FunnelRepository) UpdateFunnel(ctx context.Context, funnelID string, orgID string, updates map[string]interface{}) (*models.Funnel, error) {
	funnel := &models.Funnel{}

	query := r.db.NewUpdate().
		Model(funnel).
		Where("id = ?", funnelID).
		Where("organization_id = ?", orgID).
		Returning("*")

	for key, value := range updates {
		query = query.Set("? = ?", bun.Ident(key), value)
	}

	_, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return funnel, nil
}

func (r *FunnelRepository) DeleteFunnelSteps(ctx context.Context, funnelID string) error {
	_, err := r.db.NewDelete().
		Model(&models.FunnelStep{}).
		Where("funnel_id = ?", funnelID).
		Exec(ctx)
	return err
}

func (r *FunnelRepository) DeleteFunnel(ctx context.Context, funnelID string, orgID string) error {
	_, err := r.db.NewDelete().
		Model(&models.Funnel{}).
		Where("id = ?", funnelID).
		Where("organization_id = ?", orgID).
		Exec(ctx)
	return err
}

func (r *FunnelRepository) FunnelExists(ctx context.Context, funnelID string, orgID string) (bool, error) {
	return r.db.NewSelect().
		Model(&models.Funnel{}).
		Where("id = ?", funnelID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
}
