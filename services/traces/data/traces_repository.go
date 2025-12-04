package data

import (
	"context"
	"time"
	"zori/internal/storage/clickhouse"
	"zori/internal/telemetry"
	"zori/services/traces/types"
)

type TracesRepository struct {
	clickDb *clickhouse.ClickhouseDB
	logger  *telemetry.Logger
}

func NewTracesRepository(clickDb *clickhouse.ClickhouseDB, logger *telemetry.Logger) *TracesRepository {
	return &TracesRepository{
		clickDb: clickDb,
		logger:  logger,
	}
}

func (r *TracesRepository) BatchInsertTraces(ctx context.Context, traces []*types.LLMTrace) error {
	if len(traces) == 0 {
		return nil
	}

	batchInsert, err := r.clickDb.Db().PrepareBatch(ctx,
		`INSERT INTO llm_traces (
			project_id, trace_id, name, user_id, session_id,
			release, version, input, output, metadata, tags, public, timestamp
		)`)
	if err != nil {
		return err
	}

	for _, trace := range traces {
		err = batchInsert.Append(
			trace.ProjectID,
			trace.TraceID,
			trace.Name,
			trace.UserID,
			trace.SessionID,
			trace.Release,
			trace.Version,
			trace.Input,
			trace.Output,
			trace.Metadata,
			trace.Tags,
			trace.Public,
			trace.Timestamp,
		)
		if err != nil {
			return err
		}
	}

	return batchInsert.Send()
}

func (r *TracesRepository) BatchInsertGenerations(ctx context.Context, generations []*types.LLMGeneration) error {
	if len(generations) == 0 {
		return nil
	}

	batchInsert, err := r.clickDb.Db().PrepareBatch(ctx,
		`INSERT INTO llm_generations (
			project_id, generation_id, trace_id, user_id, parent_observation_id, name,
			model, model_parameters, input, output,
			start_time, end_time, completion_start_time, latency_ms, time_to_first_token_ms,
			input_tokens, output_tokens, total_tokens,
			input_cost, output_cost, total_cost,
			level, status_message, metadata, prompt_name, prompt_version
		)`)
	if err != nil {
		return err
	}

	for _, gen := range generations {
		var latencyMs *uint64
		if gen.EndTime != nil {
			latency := uint64(gen.EndTime.Sub(gen.StartTime).Milliseconds())
			latencyMs = &latency
		}

		var ttftMs *uint64
		if gen.CompletionStartTime != nil {
			ttft := uint64(gen.CompletionStartTime.Sub(gen.StartTime).Milliseconds())
			ttftMs = &ttft
		}

		var endTime *time.Time
		if gen.EndTime != nil {
			endTime = gen.EndTime
		}

		err = batchInsert.Append(
			gen.ProjectID,
			gen.GenerationID,
			gen.TraceID,
			gen.UserID,
			gen.ParentObservationID,
			gen.Name,
			gen.Model,
			gen.ModelParameters,
			gen.Input,
			gen.Output,
			gen.StartTime,
			endTime,
			gen.CompletionStartTime,
			latencyMs,
			ttftMs,
			gen.InputTokens,
			gen.OutputTokens,
			gen.TotalTokens,
			gen.InputCost,
			gen.OutputCost,
			gen.TotalCost,
			gen.Level,
			gen.StatusMessage,
			gen.Metadata,
			gen.PromptName,
			gen.PromptVersion,
		)
		if err != nil {
			return err
		}
	}

	return batchInsert.Send()
}
