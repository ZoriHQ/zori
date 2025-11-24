package services

import (
	"fmt"
	"zori/internal/ctx"
	"zori/services/recordings/data"
	"zori/services/recordings/types"
)

type RecordingsService struct {
	data *data.RecordingsData
}

func NewRecordingsService(data *data.RecordingsData) *RecordingsService {
	return &RecordingsService{
		data: data,
	}
}

func (s *RecordingsService) GetRecordings(ctx *ctx.Ctx, req *types.GetRecordingsRequest) (*types.GetRecordingsResponse, error) {
	if req.Limit == 0 {
		req.Limit = 20
	}

	recordings, total, err := s.data.GetSessionRecordings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get recordings: %w", err)
	}

	return &types.GetRecordingsResponse{
		Recordings: recordings,
		Total:      total,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}, nil
}

func (s *RecordingsService) GetRecordingEvents(ctx *ctx.Ctx, req *types.GetRecordingEventsRequest) (*types.SessionRecordingDetail, error) {
	detail, err := s.data.GetSessionRecordingEvents(ctx, req.ProjectID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recording events: %w", err)
	}

	return detail, nil
}
