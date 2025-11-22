package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type LLMProvider string

const (
	ProviderOpenAI    LLMProvider = "openai"
	ProviderAnthropic LLMProvider = "anthropic"
	ProviderGoogle    LLMProvider = "google"
	ProviderMistral   LLMProvider = "mistral"
	ProviderCohere    LLMProvider = "cohere"
	ProviderOpenRouter LLMProvider = "openrouter"
	ProviderCustom    LLMProvider = "custom"
)

type LLMTraceRequest struct {
	VisitorID    *string `json:"visitor_id,omitempty"`
	CustomerID   *string `json:"customer_id,omitempty"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	InputTokens  *uint32 `json:"input_tokens,omitempty"`
	OutputTokens *uint32 `json:"output_tokens,omitempty"`
	InputPrompt  *string `json:"input_prompt,omitempty"`
	OutputPrompt *string `json:"output_prompt,omitempty"`
	ExtraFee     *string `json:"extra_fee,omitempty"`
	Timestamp    *int64  `json:"timestamp,omitempty"`
}

type LLMTraceFrame struct {
	*LLMTraceRequest
	ProjectID      string `json:"project_id"`
	OrganizationID string `json:"organization_id"`
}

type LLMTrace struct {
	TraceID            string          `ch:"trace_id"`
	VisitorID          *string         `ch:"visitor_id"`
	CustomerID         *string         `ch:"customer_id"`
	Provider           string          `ch:"provider"`
	Model              string          `ch:"model"`
	InputTokens        uint32          `ch:"input_tokens"`
	OutputTokens       uint32          `ch:"output_tokens"`
	InputPrompt        *string         `ch:"input_prompt"`
	OutputPrompt       *string         `ch:"output_prompt"`
	InputCost          decimal.Decimal `ch:"input_cost"`
	OutputCost         decimal.Decimal `ch:"output_cost"`
	ExtraFee           *decimal.Decimal `ch:"extra_fee"`
	TotalCost          decimal.Decimal `ch:"total_cost"`
	ProjectID          string          `ch:"project_id"`
	OrganizationID     string          `ch:"organization_id"`
	ClientTimestampUTC time.Time       `ch:"client_timestamp_utc"`
	ServerTimestampUTC time.Time       `ch:"server_timestamp_utc"`
	CreatedAt          time.Time       `ch:"created_at"`
}

type PriceInfo struct {
	Provider          LLMProvider
	Model             string
	InputPricePerToken  decimal.Decimal
	OutputPricePerToken decimal.Decimal
	UpdatedAt         time.Time
}
