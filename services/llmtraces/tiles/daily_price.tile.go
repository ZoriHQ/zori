package tiles

import (
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"

	"github.com/shopspring/decimal"
)

type DailyPriceResponse struct {
	TotalCost         decimal.Decimal `json:"total_cost" ch:"total_cost"`
	PreviousTotalCost decimal.Decimal `json:"previous_total_cost" ch:"previous_total_cost"`
	TraceCount        uint64          `json:"trace_count" ch:"trace_count"`
	TotalInputTokens  uint64          `json:"total_input_tokens" ch:"total_input_tokens"`
	TotalOutputTokens uint64          `json:"total_output_tokens" ch:"total_output_tokens"`
}

type DailyPriceTile struct {
	db *clickhouse.ClickhouseDB
}

func NewDailyPriceTile(db *clickhouse.ClickhouseDB) *DailyPriceTile {
	return &DailyPriceTile{db: db}
}

func (t *DailyPriceTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*DailyPriceResponse, error) {
	query := `
		WITH
			current_period AS (
				SELECT
					sum(total_cost) AS total_cost,
					count() AS trace_count,
					sum(input_tokens) AS total_input_tokens,
					sum(output_tokens) AS total_output_tokens
				FROM llm_traces
				WHERE organization_id = ?
				AND project_id = ?
				AND client_timestamp_utc >= now() - ` + filter.TimeRange.IntervalValue + `
			),
			previous_period AS (
				SELECT
					sum(total_cost) AS total_cost
				FROM llm_traces
				WHERE organization_id = ?
				AND project_id = ?
				AND client_timestamp_utc >= now() - ` + filter.TimeRange.IntervalValueDelta + `
				AND client_timestamp_utc < now() - ` + filter.TimeRange.IntervalValue + `
			)
		SELECT
			current_period.total_cost AS total_cost,
			previous_period.total_cost AS previous_total_cost,
			current_period.trace_count AS trace_count,
			current_period.total_input_tokens AS total_input_tokens,
			current_period.total_output_tokens AS total_output_tokens
		FROM current_period, previous_period
	`

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)

	var data DailyPriceResponse
	if err := row.ScanStruct(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

type DailyPriceTimelineData struct {
	Date       string          `json:"date" ch:"date"`
	TotalCost  decimal.Decimal `json:"total_cost" ch:"total_cost"`
	TraceCount uint64          `json:"trace_count" ch:"trace_count"`
}

type DailyPriceTimelineResponse struct {
	Data []DailyPriceTimelineData `json:"data"`
}

type DailyPriceTimelineTile struct {
	db *clickhouse.ClickhouseDB
}

func NewDailyPriceTimelineTile(db *clickhouse.ClickhouseDB) *DailyPriceTimelineTile {
	return &DailyPriceTimelineTile{db: db}
}

func (t *DailyPriceTimelineTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*DailyPriceTimelineResponse, error) {
	query := `
		SELECT
			formatDateTime(` + filter.TimeRange.TimeBucketFunction + `(client_timestamp_utc), '%Y-%m-%d %H:%i') AS date,
			sum(total_cost) AS total_cost,
			count() AS trace_count
		FROM llm_traces
		WHERE organization_id = ?
		AND project_id = ?
		AND client_timestamp_utc >= now() - ` + filter.TimeRange.IntervalValue + `
		GROUP BY date
		ORDER BY date ASC
	`

	rows, err := t.db.Db().Query(ctx, query, ctx.OrgID(), filter.ProjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []DailyPriceTimelineData
	for rows.Next() {
		var d DailyPriceTimelineData
		if err := rows.ScanStruct(&d); err != nil {
			return nil, err
		}
		data = append(data, d)
	}

	return &DailyPriceTimelineResponse{Data: data}, nil
}
