package types

import "encoding/json"

type OtelTraceRequest struct {
	ResourceSpans []OtelResourceSpan `json:"resourceSpans"`
}

type OtelResourceSpan struct {
	Resource   *OtelResource   `json:"resource,omitempty"`
	ScopeSpans []OtelScopeSpan `json:"scopeSpans"`
}

type OtelResource struct {
	Attributes []OtelAttribute `json:"attributes,omitempty"`
}

type OtelScopeSpan struct {
	Scope *OtelScope `json:"scope,omitempty"`
	Spans []OtelSpan `json:"spans"`
}

type OtelScope struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type OtelSpan struct {
	TraceID           string          `json:"traceId"`
	SpanID            string          `json:"spanId"`
	ParentSpanID      string          `json:"parentSpanId,omitempty"`
	Name              string          `json:"name"`
	Kind              string          `json:"kind,omitempty"` // SPAN_KIND_INTERNAL, SPAN_KIND_SERVER, etc.
	StartTimeUnixNano int64           `json:"startTimeUnixNano"`
	EndTimeUnixNano   int64           `json:"endTimeUnixNano,omitempty"`
	Attributes        []OtelAttribute `json:"attributes,omitempty"`
	Status            *OtelStatus     `json:"status,omitempty"`
	Events            []OtelEvent     `json:"events,omitempty"`
}

type OtelAttribute struct {
	Key   string        `json:"key"`
	Value OtelAttrValue `json:"value"`
}

type OtelAttrValue struct {
	StringValue string          `json:"stringValue,omitempty"`
	IntValue    json.Number     `json:"intValue,omitempty"`
	DoubleValue float64         `json:"doubleValue,omitempty"`
	BoolValue   bool            `json:"boolValue,omitempty"`
	ArrayValue  *OtelArrayValue `json:"arrayValue,omitempty"`
}

type OtelArrayValue struct {
	Values []OtelAttrValue `json:"values,omitempty"`
}

type OtelStatus struct {
	Code    string `json:"code,omitempty"` // STATUS_CODE_OK, STATUS_CODE_ERROR, STATUS_CODE_UNSET
	Message string `json:"message,omitempty"`
}

type OtelEvent struct {
	Name         string          `json:"name"`
	TimeUnixNano int64           `json:"timeUnixNano"`
	Attributes   []OtelAttribute `json:"attributes,omitempty"`
}

type OtelTraceResponse struct {
	// Empty response as per Langfuse spec
}

const (
	// Model attributes
	AttrGenAIRequestModel  = "gen_ai.request.model"
	AttrGenAIResponseModel = "gen_ai.response.model"
	AttrLLMModelName       = "llm.model_name"
	AttrLangfuseModelName  = "langfuse.observation.model.name"

	// Usage attributes
	AttrGenAIUsageInputTokens      = "gen_ai.usage.input_tokens"
	AttrGenAIUsageOutputTokens     = "gen_ai.usage.output_tokens"
	AttrGenAIUsagePromptTokens     = "gen_ai.usage.prompt_tokens"
	AttrGenAIUsageCompletionTokens = "gen_ai.usage.completion_tokens"
	AttrLLMTokenCountPrompt        = "llm.token_count.prompt"
	AttrLLMTokenCountCompletion    = "llm.token_count.completion"

	// Cost attributes
	AttrGenAIUsageCost = "gen_ai.usage.cost"

	// Input/Output attributes
	AttrGenAIPrompt         = "gen_ai.prompt"
	AttrGenAICompletion     = "gen_ai.completion"
	AttrLangfuseInput       = "langfuse.observation.input"
	AttrLangfuseOutput      = "langfuse.observation.output"
	AttrLangfuseTraceInput  = "langfuse.trace.input"
	AttrLangfuseTraceOutput = "langfuse.trace.output"

	// Request parameters
	AttrGenAIRequestTemperature = "gen_ai.request.temperature"
	AttrGenAIRequestMaxTokens   = "gen_ai.request.max_tokens"
	AttrGenAIRequestTopP        = "gen_ai.request.top_p"

	// Langfuse-specific attributes
	AttrLangfuseTraceID          = "langfuse.trace.id"
	AttrLangfuseSessionID        = "langfuse.session.id"
	AttrLangfuseUserID           = "langfuse.user.id"
	AttrLangfuseTraceName        = "langfuse.trace.name"
	AttrLangfuseSpanName         = "langfuse.span.name"
	AttrLangfuseObservationType  = "langfuse.observation.type"
	AttrLangfuseObservationLevel = "langfuse.observation.level"

	// OpenRouter specific
	AttrOpenRouterModel = "openrouter.model"
)
