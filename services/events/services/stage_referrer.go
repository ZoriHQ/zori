package services

import (
	"net/url"
	"zori/services/ingestion/types"

	"github.com/Cleverse/go-utilities/nullable"
)

type StageReferrer struct {
}

func NewStageReferrer() StageReferrer {
	return StageReferrer{}
}

// ProcessFrame for StageReferrer processed referrer information and extracts path and domain for better indexing
func (s StageReferrer) ProcessFrame(event *types.ClientEventFrameV1) error {
	if event.Referrer == "" {
		return nil
	}

	parsedReferrerURL, err := url.Parse(event.Referrer)
	if err != nil {
		return err
	}

	event.ReferredDomain = nullable.FromString(parsedReferrerURL.Host).Ptr()
	event.ReferrerPath = nullable.FromString(parsedReferrerURL.Path).Ptr()

	return nil
}
