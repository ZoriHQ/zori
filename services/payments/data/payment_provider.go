package data

import (
	"context"
	"time"
	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"

	"github.com/uptrace/bun"
)

type PaymentProviderData struct {
	db *bun.DB
}

func NewPaymentProviderData(db *postgres.PostgresDB) *PaymentProviderData {
	return &PaymentProviderData{db: db.DB}
}

func (p *PaymentProviderData) CreateProvider(ctx context.Context, provider *models.PaymentProvider) error {
	_, err := p.db.NewInsert().
		Model(provider).
		Returning("*").
		Exec(ctx)
	return err
}

func (p *PaymentProviderData) GetProvider(ctx context.Context, providerID string, orgID string) (*models.PaymentProvider, error) {
	provider := &models.PaymentProvider{}
	err := p.db.NewSelect().
		Model(provider).
		Where("id = ?", providerID).
		Where("organization_id = ?", orgID).
		Scan(ctx)
	return provider, err
}

func (p *PaymentProviderData) GetProviderByAccountID(ctx context.Context, accountID string) (*models.PaymentProvider, error) {
	provider := &models.PaymentProvider{}
	err := p.db.NewSelect().
		Model(provider).
		Where("account_id = ?", accountID).
		Scan(ctx)
	return provider, err
}

func (p *PaymentProviderData) GetProviderByProjectAndType(ctx context.Context, projectID string, providerType models.ProviderType) (*models.PaymentProvider, error) {
	provider := &models.PaymentProvider{}
	err := p.db.NewSelect().
		Model(provider).
		Where("project_id = ?", projectID).
		Where("provider_type = ?", providerType).
		Where("is_active = ?", true).
		Scan(ctx)
	return provider, err
}

func (p *PaymentProviderData) ListProvidersByProject(ctx context.Context, projectID string, orgID string) ([]*models.PaymentProvider, error) {
	var providers []*models.PaymentProvider
	err := p.db.NewSelect().
		Model(&providers).
		Where("project_id = ?", projectID).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(ctx)
	return providers, err
}

func (p *PaymentProviderData) ListProvidersByOrganization(ctx context.Context, orgID string) ([]*models.PaymentProvider, error) {
	var providers []*models.PaymentProvider
	err := p.db.NewSelect().
		Model(&providers).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(ctx)
	return providers, err
}

func (p *PaymentProviderData) UpdateProvider(ctx context.Context, providerID string, orgID string, updates map[string]interface{}) (*models.PaymentProvider, error) {
	provider := &models.PaymentProvider{}

	query := p.db.NewUpdate().
		Model(provider).
		Where("id = ?", providerID).
		Where("organization_id = ?", orgID).
		Returning("*")

	for key, value := range updates {
		query = query.Set("? = ?", bun.Ident(key), value)
	}

	_, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return provider, nil
}

func (p *PaymentProviderData) UpdateLastSyncedAt(ctx context.Context, providerID string) error {
	_, err := p.db.NewUpdate().
		Model(&models.PaymentProvider{}).
		Set("last_synced_at = ?", time.Now()).
		Where("id = ?", providerID).
		Exec(ctx)
	return err
}

func (p *PaymentProviderData) DeleteProvider(ctx context.Context, providerID string, orgID string) error {
	_, err := p.db.NewDelete().
		Model(&models.PaymentProvider{}).
		Where("id = ?", providerID).
		Where("organization_id = ?", orgID).
		Exec(ctx)
	return err
}

func (p *PaymentProviderData) ProviderExists(ctx context.Context, providerID string, orgID string) (bool, error) {
	return p.db.NewSelect().
		Model(&models.PaymentProvider{}).
		Where("id = ?", providerID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
}
