package services

import (
	"zori/services/events/classifier"
	"zori/services/ingestion/types"
)

type StageClickClassification struct{}

func NewStageClickClassification() *StageClickClassification {
	return &StageClickClassification{}
}

func (s *StageClickClassification) ProcessFrame(eventFrame *types.ClientEventFrameV1) error {
	if eventFrame.ClickElement == nil {
		return nil
	}

	c := classifier.NewClassifier(eventFrame.Host)

	classification := c.Classify(
		eventFrame.ClickElement.Tag,
		eventFrame.ClickElement.Selector,
		eventFrame.ClickElement.Text,
	)

	elementType := string(classification.ElementType)
	elementCategory := string(classification.ElementCategory)

	eventFrame.ClickElementType = &elementType
	eventFrame.ClickElementCategory = &elementCategory
	eventFrame.IsCTAClick = &classification.IsCTAClick
	eventFrame.LinkDestination = classification.LinkDestination
	eventFrame.IsExternalLink = &classification.IsExternalLink
	eventFrame.IsDownloadLink = &classification.IsDownloadLink

	return nil
}
