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
	Date              time.Time       `ch:"date"`
	Provider          string          `ch:"provider"`
	Model             string          `ch:"model"`
	TotalInputTokens  uint64          `ch:"total_input_tokens"`
	TotalOutputTokens uint64          `ch:"total_output_tokens"`
	TotalCost         decimal.Decimal `ch:"total_cost"`
	TraceCount        uint64          `ch:"trace_count"`
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

func (r *LLMTraceRepository) GetTraces(ctx context.Context, orgID, projectID string, from, to time.Time, req *types.LLMTracesListRequest) ([]types.LLMTraceListItem, uint64, error) {
	whereConditions := []string{
		"organization_id = ?",
		"project_id = ?",
		"client_timestamp_utc >= ?",
		"client_timestamp_utc <= ?",
	}
	args := []interface{}{orgID, projectID, from, to}

	if req.CustomerID != nil && *req.CustomerID != "" {
		whereConditions = append(whereConditions, "customer_id = ?")
		args = append(args, *req.CustomerID)
	}

	if req.VisitorID != nil && *req.VisitorID != "" {
		whereConditions = append(whereConditions, "visitor_id = ?")
		args = append(args, *req.VisitorID)
	}

	if req.Provider != nil && *req.Provider != "" {
		whereConditions = append(whereConditions, "provider = ?")
		args = append(args, *req.Provider)
	}

	if req.Model != nil && *req.Model != "" {
		whereConditions = append(whereConditions, "model = ?")
		args = append(args, *req.Model)
	}

	whereClause := "WHERE " + whereConditions[0]
	for i := 1; i < len(whereConditions); i++ {
		whereClause += " AND " + whereConditions[i]
	}

	countQuery := `
		SELECT count()
		FROM llm_traces
		` + whereClause

	var totalCount uint64
	countRow := r.db.Db().QueryRow(ctx, countQuery, args...)
	if err := countRow.Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	if totalCount == 0 {
		return []types.LLMTraceListItem{}, 0, nil
	}

	idsQuery := `
		SELECT trace_id
		FROM llm_traces
		` + whereClause + `
		ORDER BY client_timestamp_utc DESC
		LIMIT ? OFFSET ?
	`

	idsArgs := make([]interface{}, len(args), len(args)+2)
	copy(idsArgs, args)
	idsArgs = append(idsArgs, req.Limit, req.Offset)

	idsRows, err := r.db.Db().Query(ctx, idsQuery, idsArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer idsRows.Close()

	var traceIDs []string
	for idsRows.Next() {
		var traceID string
		if err := idsRows.Scan(&traceID); err != nil {
			return nil, 0, err
		}
		traceIDs = append(traceIDs, traceID)
	}

	if len(traceIDs) == 0 {
		return []types.LLMTraceListItem{}, totalCount, nil
	}

	dataQuery := `
		SELECT
			trace_id,
			visitor_id,
			customer_id,
			provider,
			model,
			input_tokens,
			output_tokens,
			input_prompt,
			output_prompt,
			input_cost,
			output_cost,
			extra_fee,
			total_cost,
			client_timestamp_utc
		FROM llm_traces
		WHERE trace_id IN (` + clickhouse.BuildPlaceholders(len(traceIDs)) + `)
		ORDER BY client_timestamp_utc DESC
	`

	dataArgs := make([]interface{}, len(traceIDs))
	for i, id := range traceIDs {
		dataArgs[i] = id
	}

	dataRows, err := r.db.Db().Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer dataRows.Close()

	var traces []types.LLMTraceListItem
	for dataRows.Next() {
		var trace types.LLMTraceListItem
		if err := dataRows.ScanStruct(&trace); err != nil {
			return nil, 0, err
		}
		traces = append(traces, trace)
	}

	return traces, totalCount, nil
}

func (r *LLMTraceRepository) GetFilterOptions(ctx context.Context, orgID, projectID string, from, to time.Time) (*types.LLMTraceFilterOptions, error) {
	whereClause := `
		WHERE organization_id = ?
		AND project_id = ?
		AND client_timestamp_utc >= ?
		AND client_timestamp_utc <= ?
	`
	args := []interface{}{orgID, projectID, from, to}

	providersQuery := `
		SELECT DISTINCT provider
		FROM llm_traces
		` + whereClause + `
		ORDER BY provider
		LIMIT 100
	`

	providerRows, err := r.db.Db().Query(ctx, providersQuery, args...)
	if err != nil {
		return nil, err
	}
	defer providerRows.Close()

	var providers []string
	for providerRows.Next() {
		var provider string
		if err := providerRows.Scan(&provider); err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}

	modelsQuery := `
		SELECT DISTINCT model
		FROM llm_traces
		` + whereClause + `
		ORDER BY model
		LIMIT 500
	`

	modelRows, err := r.db.Db().Query(ctx, modelsQuery, args...)
	if err != nil {
		return nil, err
	}
	defer modelRows.Close()

	var models []string
	for modelRows.Next() {
		var model string
		if err := modelRows.Scan(&model); err != nil {
			return nil, err
		}
		models = append(models, model)
	}

	customerIDsQuery := `
		SELECT DISTINCT customer_id
		FROM llm_traces
		` + whereClause + `
		AND customer_id IS NOT NULL
		ORDER BY customer_id
		LIMIT 1000
	`

	customerIDRows, err := r.db.Db().Query(ctx, customerIDsQuery, args...)
	if err != nil {
		return nil, err
	}
	defer customerIDRows.Close()

	var customerIDs []string
	for customerIDRows.Next() {
		var customerID string
		if err := customerIDRows.Scan(&customerID); err != nil {
			return nil, err
		}
		customerIDs = append(customerIDs, customerID)
	}

	if providers == nil {
		providers = []string{}
	}
	if models == nil {
		models = []string{}
	}
	if customerIDs == nil {
		customerIDs = []string{}
	}

	return &types.LLMTraceFilterOptions{
		Providers:   providers,
		Models:      models,
		CustomerIDs: customerIDs,
	}, nil
}
