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

// hashEmail creates a SHA256 hash of the email for privacy-safe matching
func hashEmail(email string) string {
	hash := sha256.Sum256([]byte(email))
	return hex.EncodeToString(hash[:])
}

// UpsertVisitor creates or updates a visitor's identity information
func (r *VisitorRepository) UpsertVisitor(ctx context.Context, visitor *models.Visitor) error {
	now := time.Now()

	// If email is provided, generate hash
	if visitor.Email != nil && *visitor.Email != "" {
		emailHash := hashEmail(*visitor.Email)
		visitor.EmailHash = &emailHash
	}

	// Check if visitor already exists
	existingVisitor := &models.Visitor{}
	err := r.db.NewSelect().
		Model(existingVisitor).
		Where("visitor_id = ?", visitor.VisitorID).
		Scan(ctx)

	if err != nil {
		// Visitor doesn't exist, create new one
		visitor.FirstIdentifiedAt = &now
		visitor.LastIdentifiedAt = &now
		visitor.CreatedAt = now
		visitor.UpdatedAt = now

		_, err = r.db.NewInsert().
			Model(visitor).
			Exec(ctx)
		return err
	}

	// Visitor exists, update it
	visitor.LastIdentifiedAt = &now
	visitor.UpdatedAt = now

	// Preserve first_identified_at from existing record
	if existingVisitor.FirstIdentifiedAt != nil {
		visitor.FirstIdentifiedAt = existingVisitor.FirstIdentifiedAt
	}

	// Merge custom traits (new traits override old ones)
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

// GetVisitorByID retrieves a visitor by their visitor_id
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

// GetVisitorByUserID retrieves a visitor by their user_id
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

// GetVisitorByEmail retrieves a visitor by their email
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

// GetVisitorsByProjectID retrieves all identified visitors for a project
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
