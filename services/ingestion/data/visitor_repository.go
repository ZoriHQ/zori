package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"zori/internal/storage/postgres"
	"zori/internal/storage/postgres/models"

	"github.com/uptrace/bun"
)

type VisitorRepository struct {
	db *bun.DB
}

func NewVisitorRepository(db *postgres.PostgresDB) *VisitorRepository {
	return &VisitorRepository{db: db.DB}
}

func hashEmail(email string) string {
	hash := sha256.Sum256([]byte(email))
	return hex.EncodeToString(hash[:])
}

func (r *VisitorRepository) UpsertVisitor(ctx context.Context, visitor *models.Visitor) error {
	now := time.Now()

	if visitor.Email != nil && *visitor.Email != "" {
		emailHash := hashEmail(*visitor.Email)
		visitor.EmailHash = &emailHash
	}

	existingVisitor := &models.Visitor{}
	err := r.db.NewSelect().
		Model(existingVisitor).
		Where("visitor_id = ?", visitor.VisitorID).
		Scan(ctx)

	if err != nil {
		visitor.FirstIdentifiedAt = &now
		visitor.LastIdentifiedAt = &now
		visitor.CreatedAt = now
		visitor.UpdatedAt = now

		_, err = r.db.NewInsert().
			Model(visitor).
			Exec(ctx)
		return err
	}

	visitor.LastIdentifiedAt = &now
	visitor.UpdatedAt = now

	if existingVisitor.FirstIdentifiedAt != nil {
		visitor.FirstIdentifiedAt = existingVisitor.FirstIdentifiedAt
	}

	if existingVisitor.CustomTraits != nil && visitor.CustomTraits != nil {
		for k, v := range existingVisitor.CustomTraits {
			if _, exists := visitor.CustomTraits[k]; !exists {
				visitor.CustomTraits[k] = v
			}
		}
	}

	_, err = r.db.NewUpdate().
		Model(visitor).
		WherePK().
		Exec(ctx)

	return err
}

func (r *VisitorRepository) GetVisitorByID(ctx context.Context, visitorID string) (*models.Visitor, error) {
	visitor := &models.Visitor{}
	err := r.db.NewSelect().
		Model(visitor).
		Where("visitor_id = ?", visitorID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return visitor, nil
}

func (r *VisitorRepository) GetVisitorByUserID(ctx context.Context, projectID, userID string) (*models.Visitor, error) {
	visitor := &models.Visitor{}
	err := r.db.NewSelect().
		Model(visitor).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return visitor, nil
}

func (r *VisitorRepository) GetVisitorByEmail(ctx context.Context, projectID, email string) (*models.Visitor, error) {
	visitor := &models.Visitor{}
	err := r.db.NewSelect().
		Model(visitor).
		Where("project_id = ? AND email = ?", projectID, email).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return visitor, nil
}

func (r *VisitorRepository) GetVisitorsByProjectID(ctx context.Context, projectID string, limit int, offset int) ([]*models.Visitor, error) {
	var visitors []*models.Visitor
	err := r.db.NewSelect().
		Model(&visitors).
		Where("project_id = ?", projectID).
		Order("last_identified_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return visitors, nil
}

func (r *VisitorRepository) GetVisitorsByIDs(ctx context.Context, visitorIDs []string) ([]*models.Visitor, error) {
	if len(visitorIDs) == 0 {
		return []*models.Visitor{}, nil
	}

	var visitors []*models.Visitor
	err := r.db.NewSelect().
		Model(&visitors).
		Where("visitor_id IN (?)", bun.In(visitorIDs)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return visitors, nil
}

func (r *VisitorRepository) GetAllVisitorIDsByIdentities(ctx context.Context, projectID string, visitorIDs []string) ([]string, error) {
	if len(visitorIDs) == 0 {
		return []string{}, nil
	}

	var identities []*models.Visitor
	err := r.db.NewSelect().
		Model(&identities).
		Where("visitor_id IN (?)", bun.In(visitorIDs)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	var userIDs []string
	var externalIDs []string
	var emails []string

	for _, identity := range identities {
		if identity.UserID != nil && *identity.UserID != "" {
			userIDs = append(userIDs, *identity.UserID)
		}
		if identity.ExternalID != nil && *identity.ExternalID != "" {
			externalIDs = append(externalIDs, *identity.ExternalID)
		}
		if identity.Email != nil && *identity.Email != "" {
			emails = append(emails, *identity.Email)
		}
	}

	query := r.db.NewSelect().
		Model((*models.Visitor)(nil)).
		Column("visitor_id").
		Where("project_id = ?", projectID)

	hasCondition := false
	if len(userIDs) > 0 {
		query = query.WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("user_id IN (?)", bun.In(userIDs))
		})
		hasCondition = true
	}
	if len(externalIDs) > 0 {
		if hasCondition {
			query = query.WhereOr("external_id IN (?)", bun.In(externalIDs))
		} else {
			query = query.WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("external_id IN (?)", bun.In(externalIDs))
			})
			hasCondition = true
		}
	}
	if len(emails) > 0 {
		if hasCondition {
			query = query.WhereOr("email IN (?)", bun.In(emails))
		} else {
			query = query.WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("email IN (?)", bun.In(emails))
			})
		}
	}

	var allVisitorIDs []string
	err = query.Scan(ctx, &allVisitorIDs)
	if err != nil {
		return nil, err
	}

	return allVisitorIDs, nil
}
