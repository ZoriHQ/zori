package models

import (
	"time"

	"github.com/uptrace/go-clickhouse/ch"
)

type SessionRecording struct {
	ch.CHModel `ch:"session_recordings,partition:toYYYYMM(client_timestamp_utc),order:organization_id,order:project_id,order:session_id,order:client_timestamp_utc,order:chunk_index"`

	ID string `ch:"id"`

	VisitorID string `ch:"visitor_id"`
	SessionID string `ch:"session_id"`

	ProjectID      string `ch:"project_id"`
	OrganizationID string `ch:"organization_id"`

	PageURL string `ch:"page_url"`
	Host    string `ch:"host"`

	Events     string `ch:"events"`
	EventCount uint32 `ch:"event_count"`

	ChunkIndex   uint32 `ch:"chunk_index"`
	IsFinalChunk uint8  `ch:"is_final_chunk"`

	ClientTimestampUTC time.Time `ch:"client_timestamp_utc"`
	ServerTimestampUTC time.Time `ch:"server_timestamp_utc"`

	UserAgent string `ch:"user_agent"`
	IP        string `ch:"ip"`

	CreatedAt time.Time `ch:"created_at,type:DateTime,default:now()"`
}
