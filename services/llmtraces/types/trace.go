package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type LLMProvider string

const (
	ProviderOpenAI     LLMProvider = "openai"
	ProviderAnthropic  LLMProvider = "anthropic"
	ProviderGoogle     LLMProvider = "google"
	ProviderMistral    LLMProvider = "mistral"
	ProviderCohere     LLMProvider = "cohere"
	ProviderOpenRouter LLMProvider = "openrouter"
	ProviderCustom     LLMProvider = "custom"
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
	TraceID            string           `ch:"trace_id"`
	VisitorID          *string          `ch:"visitor_id"`
	CustomerID         *string          `ch:"customer_id"`
	Provider           string           `ch:"provider"`
	Model              string           `ch:"model"`
	InputTokens        uint32           `ch:"input_tokens"`
	OutputTokens       uint32           `ch:"output_tokens"`
	InputPrompt        *string          `ch:"input_prompt"`
	OutputPrompt       *string          `ch:"output_prompt"`
	InputCost          decimal.Decimal  `ch:"input_cost"`
	OutputCost         decimal.Decimal  `ch:"output_cost"`
	ExtraFee           *decimal.Decimal `ch:"extra_fee"`
	TotalCost          decimal.Decimal  `ch:"total_cost"`
	ProjectID          string           `ch:"project_id"`
	OrganizationID     string           `ch:"organization_id"`
	ClientTimestampUTC time.Time        `ch:"client_timestamp_utc"`
	ServerTimestampUTC time.Time        `ch:"server_timestamp_utc"`
	CreatedAt          time.Time        `ch:"created_at"`
}

type PriceInfo struct {
	Provider            LLMProvider
	Model               string
	InputPricePerToken  decimal.Decimal
	OutputPricePerToken decimal.Decimal
	UpdatedAt           time.Time
}

type LLMTracesListRequest struct {
	ProjectID      string  `query:"project_id" validate:"required"`
	TimeBoundaries string  `query:"time_boundaries"`
	CustomerID     *string `query:"customer_id"`
	VisitorID      *string `query:"visitor_id"`
	Provider       *string `query:"provider"`
	Model          *string `query:"model"`
	Limit          int     `query:"limit"`
	Offset         int     `query:"offset"`
}

type LLMTraceListItem struct {
	TraceID            string           `json:"trace_id" ch:"trace_id"`
	VisitorID          *string          `json:"visitor_id,omitempty" ch:"visitor_id"`
	CustomerID         *string          `json:"customer_id,omitempty" ch:"customer_id"`
	Provider           string           `json:"provider" ch:"provider"`
	Model              string           `json:"model" ch:"model"`
	InputTokens        uint32           `json:"input_tokens" ch:"input_tokens"`
	OutputTokens       uint32           `json:"output_tokens" ch:"output_tokens"`
	InputPrompt        *string          `json:"input_prompt,omitempty" ch:"input_prompt"`
	OutputPrompt       *string          `json:"output_prompt,omitempty" ch:"output_prompt"`
	InputCost          decimal.Decimal  `json:"input_cost" ch:"input_cost"`
	OutputCost         decimal.Decimal  `json:"output_cost" ch:"output_cost"`
	ExtraFee           *decimal.Decimal `json:"extra_fee,omitempty" ch:"extra_fee"`
	TotalCost          decimal.Decimal  `json:"total_cost" ch:"total_cost"`
	ClientTimestampUTC time.Time        `json:"client_timestamp_utc" ch:"client_timestamp_utc"`
}

type LLMTracesListResponse struct {
	Traces     []LLMTraceListItem `json:"traces"`
	TotalCount uint64             `json:"total_count"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

type LLMTraceFilterOptions struct {
	Providers   []string `json:"providers"`
	Models      []string `json:"models"`
	CustomerIDs []string `json:"customer_ids"`
}
