package services

import (
	"net/url"
	"zori/services/ingestion/types"

	"github.com/Cleverse/go-utilities/nullable"
)

type StagePage struct {
}

func NewStagePage() StagePage {
	return StagePage{}
}

// ProcessFrame for StagePage parses the request and splits URL into path
func (s StagePage) ProcessFrame(event *types.ClientEventFrameV1) error {
	if event.PageURL == "" {
		event.PagePath = nullable.FromString("/").Ptr()
		return nil
	}

	parsedPageURL, err := url.Parse(event.PageURL)
	if err != nil {
		return err
	}

	event.PagePath = nullable.FromString(parsedPageURL.Path).Ptr()

	return nil
}
