package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"zori/internal/telemetry"
	"zori/services/traces/data"
	"zori/services/traces/types"
)

type IngestionService struct {
	tracesRepo *data.TracesRepository
	logger     *telemetry.Logger
}

func NewIngestionService(tracesRepo *data.TracesRepository, logger *telemetry.Logger) *IngestionService {
	return &IngestionService{
		tracesRepo: tracesRepo,
		logger:     logger,
	}
}

func (s *IngestionService) ProcessBatch(ctx context.Context, projectID string, events []types.IngestionEvent) *types.IngestionResponse {
	response := &types.IngestionResponse{
		Successes: make([]types.IngestionSuccess, 0, len(events)),
		Errors:    make([]types.IngestionError, 0),
	}

	traces := make([]*types.LLMTrace, 0)
	generations := make([]*types.LLMGeneration, 0)

	for _, event := range events {
		switch event.Type {
		case "trace-create", "trace-update":
			trace, err := s.parseTrace(projectID, event)
			if err != nil {
				response.Errors = append(response.Errors, types.IngestionError{
					ID:      event.ID,
					Status:  400,
					Message: err.Error(),
					Error:   "ValidationError",
				})
				continue
			}
			traces = append(traces, trace)
			response.Successes = append(response.Successes, types.IngestionSuccess{
				ID:     event.ID,
				Status: 201,
			})

		case "generation-create", "generation-update":
			gen, err := s.parseGeneration(projectID, event)
			if err != nil {
				response.Errors = append(response.Errors, types.IngestionError{
					ID:      event.ID,
					Status:  400,
					Message: err.Error(),
					Error:   "ValidationError",
				})
				continue
			}
			generations = append(generations, gen)
			response.Successes = append(response.Successes, types.IngestionSuccess{
				ID:     event.ID,
				Status: 201,
			})

		case "span-create", "span-update", "score-create", "event-create", "sdk-log":
			response.Successes = append(response.Successes, types.IngestionSuccess{
				ID:     event.ID,
				Status: 201,
			})

		default:
			response.Errors = append(response.Errors, types.IngestionError{
				ID:      event.ID,
				Status:  400,
				Message: fmt.Sprintf("Unknown event type: %s", event.Type),
				Error:   "ValidationError",
			})
		}
	}

	if len(traces) > 0 {
		if err := s.tracesRepo.BatchInsertTraces(ctx, traces); err != nil {
			s.logger.Error("Failed to insert traces", telemetry.Error(err))
		}
	}

	if len(generations) > 0 {
		if err := s.tracesRepo.BatchInsertGenerations(ctx, generations); err != nil {
			s.logger.Error("Failed to insert generations", telemetry.Error(err))
		}
	}

	return response
}

func (s *IngestionService) parseTrace(projectID string, event types.IngestionEvent) (*types.LLMTrace, error) {
	var body types.TraceBody
	if err := json.Unmarshal(event.Body, &body); err != nil {
		return nil, fmt.Errorf("invalid trace body: %w", err)
	}

	if body.ID == "" {
		return nil, fmt.Errorf("trace id is required")
	}

	timestamp := time.Now()
	if body.Timestamp != nil {
		if t, err := time.Parse(time.RFC3339Nano, *body.Timestamp); err == nil {
			timestamp = t
		}
	}

	inputStr := "{}"
	if body.Input != nil {
		inputStr = string(body.Input)
	}

	outputStr := "{}"
	if body.Output != nil {
		outputStr = string(body.Output)
	}

	metadataStr := "{}"
	if body.Metadata != nil {
		if b, err := json.Marshal(body.Metadata); err == nil {
			metadataStr = string(b)
		}
	}

	public := false
	if body.Public != nil {
		public = *body.Public
	}

	tags := body.Tags
	if tags == nil {
		tags = []string{}
	}

	return &types.LLMTrace{
		ProjectID: projectID,
		TraceID:   body.ID,
		Name:      body.Name,
		UserID:    body.UserID,
		SessionID: body.SessionID,
		Release:   body.Release,
		Version:   body.Version,
		Input:     inputStr,
		Output:    outputStr,
		Metadata:  metadataStr,
		Tags:      tags,
		Public:    public,
		Timestamp: timestamp,
	}, nil
}

func (s *IngestionService) ProcessOtelTraces(ctx context.Context, projectID string, req *types.OtelTraceRequest) error {
	traces := make([]*types.LLMTrace, 0)
	generations := make([]*types.LLMGeneration, 0)

	for _, resourceSpan := range req.ResourceSpans {
		resourceAttrs := extractAttributes(resourceSpan.Resource)

		for _, scopeSpan := range resourceSpan.ScopeSpans {
			for _, span := range scopeSpan.Spans {
				spanAttrs := make(map[string]any)
				for _, attr := range span.Attributes {
					spanAttrs[attr.Key] = getAttrValue(attr.Value)
				}

				isGeneration := s.isLLMGeneration(spanAttrs)

				if isGeneration {
					gen := s.otelSpanToGeneration(projectID, span, spanAttrs, resourceAttrs)
					if gen != nil {
						generations = append(generations, gen)
					}
				} else {
					trace := s.otelSpanToTrace(projectID, span, spanAttrs, resourceAttrs)
					if trace != nil {
						traces = append(traces, trace)
					}
				}
			}
		}
	}

	if len(traces) > 0 {
		if err := s.tracesRepo.BatchInsertTraces(ctx, traces); err != nil {
			s.logger.Error("Failed to insert OTEL traces", telemetry.Error(err))
			return err
		}
	}

	if len(generations) > 0 {
		if err := s.tracesRepo.BatchInsertGenerations(ctx, generations); err != nil {
			s.logger.Error("Failed to insert OTEL generations", telemetry.Error(err))
			return err
		}
	}

	s.logger.Info("Processed OTEL traces",
		telemetry.Int("traces", len(traces)),
		telemetry.Int("generations", len(generations)))

	return nil
}

func (s *IngestionService) isLLMGeneration(attrs map[string]any) bool {
	if obsType, ok := attrs[types.AttrLangfuseObservationType].(string); ok && obsType == "generation" {
		return true
	}

	llmKeys := []string{
		types.AttrGenAIRequestModel,
		types.AttrGenAIResponseModel,
		types.AttrLLMModelName,
		types.AttrLangfuseModelName,
		types.AttrGenAIUsageInputTokens,
		types.AttrGenAIUsageOutputTokens,
		types.AttrOpenRouterModel,
		"gen_ai.system",
		"llm.request.type",
	}

	for _, key := range llmKeys {
		if _, ok := attrs[key]; ok {
			return true
		}
	}
	return false
}

func (s *IngestionService) otelSpanToTrace(projectID string, span types.OtelSpan, attrs map[string]any, resourceAttrs map[string]any) *types.LLMTrace {
	if span.TraceID == "" {
		return nil
	}

	timestamp := parseOtelTimestampNano(span.StartTimeUnixNano)

	var name *string
	if n, ok := attrs[types.AttrLangfuseTraceName].(string); ok && n != "" {
		name = &n
	} else if span.Name != "" {
		name = &span.Name
	}

	var sessionID *string
	if s, ok := attrs[types.AttrLangfuseSessionID].(string); ok && s != "" {
		sessionID = &s
	}

	var userID *string
	if u, ok := attrs[types.AttrLangfuseUserID].(string); ok && u != "" {
		userID = &u
	}

	metadata := make(map[string]any)
	metadata["otel_attributes"] = attrs
	metadata["otel_resource_attributes"] = resourceAttrs
	metadata["span_id"] = span.SpanID
	if span.ParentSpanID != "" {
		metadata["parent_span_id"] = span.ParentSpanID
	}

	metadataStr := "{}"
	if b, err := json.Marshal(metadata); err == nil {
		metadataStr = string(b)
	}

	inputStr := "{}"
	for _, key := range []string{types.AttrLangfuseTraceInput, types.AttrLangfuseInput, types.AttrGenAIPrompt} {
		if input, ok := attrs[key].(string); ok && input != "" {
			inputStr = input
			break
		}
	}

	outputStr := "{}"
	for _, key := range []string{types.AttrLangfuseTraceOutput, types.AttrLangfuseOutput, types.AttrGenAICompletion} {
		if output, ok := attrs[key].(string); ok && output != "" {
			outputStr = output
			break
		}
	}

	return &types.LLMTrace{
		ProjectID: projectID,
		TraceID:   span.TraceID,
		Name:      name,
		UserID:    userID,
		SessionID: sessionID,
		Input:     inputStr,
		Output:    outputStr,
		Metadata:  metadataStr,
		Tags:      []string{},
		Public:    false,
		Timestamp: timestamp,
	}
}

func (s *IngestionService) otelSpanToGeneration(projectID string, span types.OtelSpan, attrs map[string]any, resourceAttrs map[string]any) *types.LLMGeneration {
	if span.TraceID == "" || span.SpanID == "" {
		return nil
	}

	startTime := parseOtelTimestampNano(span.StartTimeUnixNano)
	var endTime *time.Time
	if span.EndTimeUnixNano > 0 {
		t := parseOtelTimestampNano(span.EndTimeUnixNano)
		endTime = &t
	}

	var model *string
	for _, key := range []string{
		types.AttrGenAIResponseModel,
		types.AttrGenAIRequestModel,
		types.AttrLLMModelName,
		types.AttrLangfuseModelName,
		types.AttrOpenRouterModel,
	} {
		if m, ok := attrs[key].(string); ok && m != "" {
			model = &m
			break
		}
	}

	var name *string
	if n, ok := attrs[types.AttrLangfuseSpanName].(string); ok && n != "" {
		name = &n
	} else if span.Name != "" {
		name = &span.Name
	}

	var inputTokens, outputTokens, totalTokens *uint32
	if v := getIntAttr(attrs, types.AttrGenAIUsageInputTokens, types.AttrGenAIUsagePromptTokens, types.AttrLLMTokenCountPrompt); v > 0 {
		t := uint32(v)
		inputTokens = &t
	}
	if v := getIntAttr(attrs, types.AttrGenAIUsageOutputTokens, types.AttrGenAIUsageCompletionTokens, types.AttrLLMTokenCountCompletion); v > 0 {
		t := uint32(v)
		outputTokens = &t
	}
	if inputTokens != nil && outputTokens != nil {
		t := *inputTokens + *outputTokens
		totalTokens = &t
	}

	var totalCost *float64
	if c, ok := attrs[types.AttrGenAIUsageCost].(float64); ok && c > 0 {
		totalCost = &c
	}

	inputStr := "{}"
	for _, key := range []string{types.AttrLangfuseInput, types.AttrGenAIPrompt} {
		if input, ok := attrs[key].(string); ok && input != "" {
			inputStr = input
			break
		}
	}

	outputStr := "{}"
	for _, key := range []string{types.AttrLangfuseOutput, types.AttrGenAICompletion} {
		if output, ok := attrs[key].(string); ok && output != "" {
			outputStr = output
			break
		}
	}

	modelParams := make(map[string]any)
	if temp, ok := attrs[types.AttrGenAIRequestTemperature]; ok {
		modelParams["temperature"] = temp
	}
	if maxTokens, ok := attrs[types.AttrGenAIRequestMaxTokens]; ok {
		modelParams["max_tokens"] = maxTokens
	}
	if topP, ok := attrs[types.AttrGenAIRequestTopP]; ok {
		modelParams["top_p"] = topP
	}
	modelParamsStr := "{}"
	if len(modelParams) > 0 {
		if b, err := json.Marshal(modelParams); err == nil {
			modelParamsStr = string(b)
		}
	}

	metadata := make(map[string]any)
	metadata["otel_attributes"] = attrs
	metadata["otel_resource_attributes"] = resourceAttrs
	metadataStr := "{}"
	if b, err := json.Marshal(metadata); err == nil {
		metadataStr = string(b)
	}

	level := "DEFAULT"
	if lvl, ok := attrs[types.AttrLangfuseObservationLevel].(string); ok && lvl != "" {
		level = lvl
	} else if span.Status != nil && span.Status.Code == "STATUS_CODE_ERROR" {
		level = "ERROR"
	}

	var statusMessage *string
	if span.Status != nil && span.Status.Message != "" {
		statusMessage = &span.Status.Message
	}

	var parentObservationID *string
	if span.ParentSpanID != "" {
		parentObservationID = &span.ParentSpanID
	}

	var userID *string
	if u, ok := attrs[types.AttrLangfuseUserID].(string); ok && u != "" {
		userID = &u
	}

	return &types.LLMGeneration{
		ProjectID:           projectID,
		GenerationID:        span.SpanID,
		TraceID:             span.TraceID,
		UserID:              userID,
		ParentObservationID: parentObservationID,
		Name:                name,
		Model:               model,
		ModelParameters:     modelParamsStr,
		Input:               inputStr,
		Output:              outputStr,
		StartTime:           startTime,
		EndTime:             endTime,
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		TotalTokens:         totalTokens,
		TotalCost:           totalCost,
		Level:               level,
		StatusMessage:       statusMessage,
		Metadata:            metadataStr,
	}
}

func extractAttributes(resource *types.OtelResource) map[string]any {
	attrs := make(map[string]any)
	if resource == nil {
		return attrs
	}
	for _, attr := range resource.Attributes {
		attrs[attr.Key] = getAttrValue(attr.Value)
	}
	return attrs
}

func getAttrValue(v types.OtelAttrValue) any {
	if v.StringValue != "" {
		return v.StringValue
	}
	if v.IntValue != "" {
		if i, err := v.IntValue.Int64(); err == nil {
			return i
		}
		return v.IntValue.String()
	}
	if v.DoubleValue != 0 {
		return v.DoubleValue
	}
	if v.BoolValue {
		return v.BoolValue
	}
	if v.ArrayValue != nil {
		arr := make([]any, len(v.ArrayValue.Values))
		for i, val := range v.ArrayValue.Values {
			arr[i] = getAttrValue(val)
		}
		return arr
	}
	return nil
}

func getIntAttr(attrs map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			switch val := v.(type) {
			case int64:
				return val
			case int:
				return int64(val)
			case float64:
				return int64(val)
			case string:
				if i, err := strconv.ParseInt(val, 10, 64); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

func parseOtelTimestampNano(nanos int64) time.Time {
	if nanos == 0 {
		return time.Now()
	}
	return time.Unix(0, nanos)
}

func (s *IngestionService) parseGeneration(projectID string, event types.IngestionEvent) (*types.LLMGeneration, error) {
	var body types.GenerationBody
	if err := json.Unmarshal(event.Body, &body); err != nil {
		return nil, fmt.Errorf("invalid generation body: %w", err)
	}

	if body.ID == "" {
		return nil, fmt.Errorf("generation id is required")
	}
	if body.TraceID == "" {
		return nil, fmt.Errorf("traceId is required")
	}

	startTime, err := time.Parse(time.RFC3339Nano, body.StartTime)
	if err != nil {
		startTime, err = time.Parse(time.RFC3339, body.StartTime)
		if err != nil {
			return nil, fmt.Errorf("invalid startTime: %w", err)
		}
	}

	var endTime *time.Time
	if body.EndTime != nil {
		if t, err := time.Parse(time.RFC3339Nano, *body.EndTime); err == nil {
			endTime = &t
		} else if t, err := time.Parse(time.RFC3339, *body.EndTime); err == nil {
			endTime = &t
		}
	}

	var completionStartTime *time.Time
	if body.CompletionStartTime != nil {
		if t, err := time.Parse(time.RFC3339Nano, *body.CompletionStartTime); err == nil {
			completionStartTime = &t
		} else if t, err := time.Parse(time.RFC3339, *body.CompletionStartTime); err == nil {
			completionStartTime = &t
		}
	}

	inputStr := "{}"
	if body.Input != nil {
		inputStr = string(body.Input)
	}

	outputStr := "{}"
	if body.Output != nil {
		outputStr = string(body.Output)
	}

	modelParamsStr := "{}"
	if body.ModelParameters != nil {
		if b, err := json.Marshal(body.ModelParameters); err == nil {
			modelParamsStr = string(b)
		}
	}

	metadataStr := "{}"
	if body.Metadata != nil {
		if b, err := json.Marshal(body.Metadata); err == nil {
			metadataStr = string(b)
		}
	}

	level := "DEFAULT"
	if body.Level != nil {
		level = *body.Level
	}

	var inputTokens, outputTokens, totalTokens *uint32
	var inputCost, outputCost, totalCost *float64

	if body.Usage != nil {
		if body.Usage.Input != nil {
			v := uint32(*body.Usage.Input)
			inputTokens = &v
		}
		if body.Usage.Output != nil {
			v := uint32(*body.Usage.Output)
			outputTokens = &v
		}
		if body.Usage.Total != nil {
			v := uint32(*body.Usage.Total)
			totalTokens = &v
		}
		inputCost = body.Usage.InputCost
		outputCost = body.Usage.OutputCost
		totalCost = body.Usage.TotalCost
	}

	var promptVersion *uint32
	if body.PromptVersion != nil {
		v := uint32(*body.PromptVersion)
		promptVersion = &v
	}

	return &types.LLMGeneration{
		ProjectID:           projectID,
		GenerationID:        body.ID,
		TraceID:             body.TraceID,
		ParentObservationID: body.ParentObservationID,
		Name:                body.Name,
		Model:               body.Model,
		ModelParameters:     modelParamsStr,
		Input:               inputStr,
		Output:              outputStr,
		StartTime:           startTime,
		EndTime:             endTime,
		CompletionStartTime: completionStartTime,
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		TotalTokens:         totalTokens,
		InputCost:           inputCost,
		OutputCost:          outputCost,
		TotalCost:           totalCost,
		Level:               level,
		StatusMessage:       body.StatusMessage,
		Metadata:            metadataStr,
		PromptName:          body.PromptName,
		PromptVersion:       promptVersion,
	}, nil
}
