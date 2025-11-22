package data

import (
	"context"
	"time"

	"zori/internal/storage/clickhouse"
	"zori/services/llmtraces/types"

	"github.com/shopspring/decimal"
)

type LLMTraceRepository struct {
	db *clickhouse.ClickhouseDB
}

func NewLLMTraceRepository(db *clickhouse.ClickhouseDB) *LLMTraceRepository {
	return &LLMTraceRepository{db: db}
}

func (r *LLMTraceRepository) Insert(ctx context.Context, trace *types.LLMTrace) error {
	query := `
		INSERT INTO llm_traces (
			trace_id, visitor_id, customer_id, provider, model,
			input_tokens, output_tokens, input_prompt, output_prompt,
			input_cost, output_cost, extra_fee, total_cost,
			project_id, organization_id, client_timestamp_utc
		) VALUES (
			generateUUIDv4(), ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?
		)
	`

	return r.db.Db().Exec(ctx, query,
		trace.VisitorID, trace.CustomerID, trace.Provider, trace.Model,
		trace.InputTokens, trace.OutputTokens, trace.InputPrompt, trace.OutputPrompt,
		trace.InputCost, trace.OutputCost, trace.ExtraFee, trace.TotalCost,
		trace.ProjectID, trace.OrganizationID, trace.ClientTimestampUTC,
	)
}

type DailyCost struct {
	Date             time.Time       `ch:"date"`
	Provider         string          `ch:"provider"`
	Model            string          `ch:"model"`
	TotalInputTokens uint64          `ch:"total_input_tokens"`
	TotalOutputTokens uint64         `ch:"total_output_tokens"`
	TotalCost        decimal.Decimal `ch:"total_cost"`
	TraceCount       uint64          `ch:"trace_count"`
}

func (r *LLMTraceRepository) GetDailyCosts(ctx context.Context, orgID, projectID string, from, to time.Time) ([]DailyCost, error) {
	query := `
		SELECT
			date,
			provider,
			model,
			sum(total_input_tokens) AS total_input_tokens,
			sum(total_output_tokens) AS total_output_tokens,
			sum(total_cost) AS total_cost,
			sum(trace_count) AS trace_count
		FROM llm_traces_daily_costs_mv
		WHERE organization_id = ?
		AND project_id = ?
		AND date >= ?
		AND date <= ?
		GROUP BY date, provider, model
		ORDER BY date DESC
	`

	rows, err := r.db.Db().Query(ctx, query, orgID, projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var costs []DailyCost
	for rows.Next() {
		var cost DailyCost
		if err := rows.ScanStruct(&cost); err != nil {
			return nil, err
		}
		costs = append(costs, cost)
	}

	return costs, nil
}

type CustomerCost struct {
	CustomerID        string          `ch:"customer_id"`
	TotalInputTokens  uint64          `ch:"total_input_tokens"`
	TotalOutputTokens uint64          `ch:"total_output_tokens"`
	TotalCost         decimal.Decimal `ch:"total_cost"`
	TraceCount        uint64          `ch:"trace_count"`
}

func (r *LLMTraceRepository) GetCustomerCosts(ctx context.Context, orgID, projectID string, from, to time.Time) ([]CustomerCost, error) {
	query := `
		SELECT
			customer_id,
			sum(total_input_tokens) AS total_input_tokens,
			sum(total_output_tokens) AS total_output_tokens,
			sum(total_cost) AS total_cost,
			sum(trace_count) AS trace_count
		FROM llm_traces_customer_costs_mv
		WHERE organization_id = ?
		AND project_id = ?
		AND date >= ?
		AND date <= ?
		GROUP BY customer_id
		ORDER BY total_cost DESC
	`

	rows, err := r.db.Db().Query(ctx, query, orgID, projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var costs []CustomerCost
	for rows.Next() {
		var cost CustomerCost
		if err := rows.ScanStruct(&cost); err != nil {
			return nil, err
		}
		costs = append(costs, cost)
	}

	return costs, nil
}

func (r *LLMTraceRepository) GetTotalCostForPeriod(ctx context.Context, orgID, projectID string, from, to time.Time) (decimal.Decimal, uint64, error) {
	query := `
		SELECT
			sum(total_cost) AS total_cost,
			count() AS trace_count
		FROM llm_traces
		WHERE organization_id = ?
		AND project_id = ?
		AND client_timestamp_utc >= ?
		AND client_timestamp_utc <= ?
	`

	row := r.db.Db().QueryRow(ctx, query, orgID, projectID, from, to)

	var totalCost decimal.Decimal
	var traceCount uint64
	if err := row.Scan(&totalCost, &traceCount); err != nil {
		return decimal.Zero, 0, err
	}

	return totalCost, traceCount, nil
}

func (r *LLMTraceRepository) GetCustomerCount(ctx context.Context, orgID, projectID string, from, to time.Time) (uint64, error) {
	query := `
		SELECT uniqExact(customer_id) AS customer_count
		FROM llm_traces
		WHERE organization_id = ?
		AND project_id = ?
		AND client_timestamp_utc >= ?
		AND client_timestamp_utc <= ?
		AND customer_id IS NOT NULL
	`

	row := r.db.Db().QueryRow(ctx, query, orgID, projectID, from, to)

	var count uint64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
