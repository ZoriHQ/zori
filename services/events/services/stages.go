package services

import "zori/services/ingestion/types"

type ProcessorStage interface {
	ProcessFrame(event *types.ClientEventFrameV1) error
}
