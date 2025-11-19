package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"zori/internal/storage/postgres/models"
	"zori/services/ingestion/data"
	"zori/services/ingestion/types"
)

type StageIdentity struct {
	visitorRepository *data.VisitorRepository
}

func NewStageIdentity() *StageIdentity {
	return &StageIdentity{}
}

func NewStageIdentityWithRepository(visitorRepository *data.VisitorRepository) *StageIdentity {
	return &StageIdentity{
		visitorRepository: visitorRepository,
	}
}

func (s *StageIdentity) ProcessFrame(frame *types.ClientEventFrameV1) error {
	// Hash email if present
	if frame.Email != nil && *frame.Email != "" {
		emailHash := hashEmail(*frame.Email)
		frame.EmailHash = &emailHash
	}

	if s.visitorRepository != nil {
		hasIdentity := (frame.UserID != nil && *frame.UserID != "") ||
			(frame.ExternalID != nil && *frame.ExternalID != "") ||
			(frame.Email != nil && *frame.Email != "")

		if hasIdentity {
			visitor := &models.Visitor{
				VisitorID:      frame.VisitorID,
				ProjectID:      frame.ProjectID,
				OrganizationID: frame.OrganizationID,
				UserID:         frame.UserID,
				ExternalID:     frame.ExternalID,
				Email:          frame.Email,
			}

			if err := s.visitorRepository.UpsertVisitor(context.Background(), visitor); err != nil {
				fmt.Println(fmt.Sprintf("Error upserting visitor: %s", err.Error()))
			}
		}
	}

	return nil
}

func hashEmail(email string) string {
	hasher := sha256.New()
	hasher.Write([]byte(email))
	return hex.EncodeToString(hasher.Sum(nil))
}
