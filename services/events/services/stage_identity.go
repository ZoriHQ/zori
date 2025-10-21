package services

import (
	"crypto/sha256"
	"encoding/hex"
	"zori/services/ingestion/types"
)

type StageIdentity struct{}

func NewStageIdentity() *StageIdentity {
	return &StageIdentity{}
}

func (s *StageIdentity) ProcessFrame(frame *types.ClientEventFrameV1) error {
	if frame.Email != nil && *frame.Email != "" {
		emailHash := hashEmail(*frame.Email)
		frame.EmailHash = &emailHash
	}

	return nil
}

func hashEmail(email string) string {
	hasher := sha256.New()
	hasher.Write([]byte(email))
	return hex.EncodeToString(hasher.Sum(nil))
}
