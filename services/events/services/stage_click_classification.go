package services

import (
	"zori/services/events/classifier"
	"zori/services/ingestion/types"
)

// StageClickClassification processes click element classification
type StageClickClassification struct{}

// NewStageClickClassification creates a new click classification processing stage
func NewStageClickClassification() *StageClickClassification {
	return &StageClickClassification{}
}

// ProcessFrame classifies click elements in the event frame
func (s *StageClickClassification) ProcessFrame(eventFrame *types.ClientEventFrameV1) error {
	// Only process events that have click element data
	if eventFrame.ClickElement == nil {
		return nil
	}

	// Create classifier with current host
	c := classifier.NewClassifier(eventFrame.Host)

	// Classify the click element
	classification := c.Classify(
		eventFrame.ClickElement.Tag,
		eventFrame.ClickElement.Selector,
		eventFrame.ClickElement.Text,
	)

	// Populate event frame with classification data
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
