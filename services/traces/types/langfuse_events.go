package types

import (
	"encoding/json"
	"time"
)

type IngestionRequest struct {
	Batch    []IngestionEvent   `json:"batch"`
	Metadata *IngestionMetadata `json:"metadata,omitempty"`
}

type IngestionMetadata struct {
	BatchSize  int    `json:"batch_size,omitempty"`
	SDKName    string `json:"sdk_name,omitempty"`
	SDKVersion string `json:"sdk_version,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
}

type IngestionEvent struct {
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"` // trace-create, generation-create, etc.
	Body      json.RawMessage `json:"body"`
}

type TraceBody struct {
	ID        string          `json:"id"`
	Timestamp *string         `json:"timestamp,omitempty"`
	Name      *string         `json:"name,omitempty"`
	UserID    *string         `json:"userId,omitempty"`
	SessionID *string         `json:"sessionId,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Output    json.RawMessage `json:"output,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
	Release   *string         `json:"release,omitempty"`
	Version   *string         `json:"version,omitempty"`
	Public    *bool           `json:"public,omitempty"`
}

type GenerationBody struct {
	ID                  string          `json:"id"`
	TraceID             string          `json:"traceId"`
	ParentObservationID *string         `json:"parentObservationId,omitempty"`
	Name                *string         `json:"name,omitempty"`
	StartTime           string          `json:"startTime"`
	EndTime             *string         `json:"endTime,omitempty"`
	CompletionStartTime *string         `json:"completionStartTime,omitempty"`
	Model               *string         `json:"model,omitempty"`
	ModelParameters     map[string]any  `json:"modelParameters,omitempty"`
	Input               json.RawMessage `json:"input,omitempty"`
	Output              json.RawMessage `json:"output,omitempty"`
	Usage               *UsageObject    `json:"usage,omitempty"`
	Metadata            map[string]any  `json:"metadata,omitempty"`
	Level               *string         `json:"level,omitempty"` // DEBUG, DEFAULT, WARNING, ERROR
	StatusMessage       *string         `json:"statusMessage,omitempty"`
	PromptName          *string         `json:"promptName,omitempty"`
	PromptVersion       *int            `json:"promptVersion,omitempty"`
}

type UsageObject struct {
	Input      *int     `json:"input,omitempty"`
	Output     *int     `json:"output,omitempty"`
	Total      *int     `json:"total,omitempty"`
	Unit       *string  `json:"unit,omitempty"` // TOKENS, CHARACTERS, etc.
	InputCost  *float64 `json:"inputCost,omitempty"`
	OutputCost *float64 `json:"outputCost,omitempty"`
	TotalCost  *float64 `json:"totalCost,omitempty"`
}

type IngestionResponse struct {
	Successes []IngestionSuccess `json:"successes"`
	Errors    []IngestionError   `json:"errors"`
}

type IngestionSuccess struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

type IngestionError struct {
	ID      string `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type LLMTrace struct {
	ProjectID string
	TraceID   string
	Name      *string
	UserID    *string
	SessionID *string
	Release   *string
	Version   *string
	Input     string
	Output    string
	Metadata  string
	Tags      []string
	Public    bool
	Timestamp time.Time
}

type LLMGeneration struct {
	ProjectID           string
	GenerationID        string
	TraceID             string
	UserID              *string
	ParentObservationID *string
	Name                *string
	Model               *string
	ModelParameters     string
	Input               string
	Output              string
	StartTime           time.Time
	EndTime             *time.Time
	CompletionStartTime *time.Time
	LatencyMs           *uint64
	TimeToFirstTokenMs  *uint64
	InputTokens         *uint32
	OutputTokens        *uint32
	TotalTokens         *uint32
	InputCost           *float64
	OutputCost          *float64
	TotalCost           *float64
	Level               string
	StatusMessage       *string
	Metadata            string
	PromptName          *string
	PromptVersion       *uint32
}
