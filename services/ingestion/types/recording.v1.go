package types

import "time"

type RecordingEventV1 struct {
	EventName              string    `json:"event_name"`
	ClientGeneratedEventID string    `json:"client_generated_event_id"`
	VisitorID              string    `json:"visitor_id"`
	SessionID              string    `json:"session_id"`
	ClientTimestampUTC     time.Time `json:"client_timestamp_utc"`
	PageURL                string    `json:"page_url"`
	Host                   string    `json:"host"`

	RecordingEvents []any  `json:"recording_events"`
	EventCount      uint32 `json:"event_count"`

	UserAgent string `json:"user_agent"`
	IP        string `json:"ip"`
}

type RecordingEventFrameV1 struct {
	*RecordingEventV1
	ProjectID      string `json:"project_id"`
	OrganizationID string `json:"organization_id"`
	ChunkIndex     uint32 `json:"chunk_index"`
	IsFinalChunk   bool   `json:"is_final_chunk"`
}
