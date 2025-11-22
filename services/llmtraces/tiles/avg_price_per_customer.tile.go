package tiles

import (
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"

	"github.com/shopspring/decimal"
)

type AvgPricePerCustomerResponse struct {
	AvgPrice         decimal.Decimal `json:"avg_price" ch:"avg_price"`
	PreviousAvgPrice decimal.Decimal `json:"previous_avg_price" ch:"previous_avg_price"`
	CustomerCount    uint64          `json:"customer_count" ch:"customer_count"`
	TotalCost        decimal.Decimal `json:"total_cost" ch:"total_cost"`
}

type AvgPricePerCustomerTile struct {
	db *clickhouse.ClickhouseDB
}

func NewAvgPricePerCustomerTile(db *clickhouse.ClickhouseDB) *AvgPricePerCustomerTile {
	return &AvgPricePerCustomerTile{db: db}
}

func (t *AvgPricePerCustomerTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*AvgPricePerCustomerResponse, error) {
	query := `
		WITH
			current_period AS (
				SELECT
					sum(total_cost) AS total_cost,
					uniqExact(customer_id) AS customer_count
				FROM llm_traces
				WHERE organization_id = ?
				AND project_id = ?
				AND client_timestamp_utc >= now() - ` + filter.TimeRange.IntervalValue + `
				AND customer_id IS NOT NULL
			),
			previous_period AS (
				SELECT
					sum(total_cost) AS total_cost,
					uniqExact(customer_id) AS customer_count
				FROM llm_traces
				WHERE organization_id = ?
				AND project_id = ?
				AND client_timestamp_utc >= now() - ` + filter.TimeRange.IntervalValueDelta + `
				AND client_timestamp_utc < now() - ` + filter.TimeRange.IntervalValue + `
				AND customer_id IS NOT NULL
			)
		SELECT
			if(current_period.customer_count > 0, current_period.total_cost / current_period.customer_count, 0) AS avg_price,
			if(previous_period.customer_count > 0, previous_period.total_cost / previous_period.customer_count, 0) AS previous_avg_price,
			current_period.customer_count AS customer_count,
			current_period.total_cost AS total_cost
		FROM current_period, previous_period
	`

	row := t.db.Db().QueryRow(ctx, query, ctx.OrgID(), filter.ProjectID, ctx.OrgID(), filter.ProjectID)

	var data AvgPricePerCustomerResponse
	if err := row.ScanStruct(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
