package types

import "time"

type GetRecordingsRequest struct {
	ProjectID string  `query:"project_id" validate:"required,uuid"`
	VisitorID *string `query:"visitor_id"`
	SessionID *string `query:"session_id"`
	Limit     int     `query:"limit" validate:"min=1,max=100"`
	Offset    int     `query:"offset" validate:"min=0"`
}

type SessionRecordingSummary struct {
	SessionID    string    `json:"session_id"`
	VisitorID    string    `json:"visitor_id"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	TotalEvents  uint64    `json:"total_events"`
	ChunkCount   uint64    `json:"chunk_count"`
	FirstPageURL string    `json:"first_page_url"`
	Host         string    `json:"host"`
	UserAgent    string    `json:"user_agent"`
}

type GetRecordingsResponse struct {
	Recordings []SessionRecordingSummary `json:"recordings"`
	Total      uint64                    `json:"total"`
	Limit      int                       `json:"limit"`
	Offset     int                       `json:"offset"`
}

type GetRecordingEventsRequest struct {
	ProjectID string `query:"project_id" validate:"required,uuid"`
	SessionID string `param:"session_id" validate:"required"`
}

type SessionRecordingDetail struct {
	SessionID string `json:"session_id"`
	VisitorID string `json:"visitor_id"`
	Host      string `json:"host"`
	UserAgent string `json:"user_agent"`
	IP        string `json:"ip"`
	Events    []any  `json:"events"`
}
