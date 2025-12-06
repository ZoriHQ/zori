package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"

	"github.com/cockroachdb/errors"
)

type LLMCostResponse struct {
	Cost         float64 `json:"cost" ch:"cost"`
	PreviousCost float64 `json:"previous_cost" ch:"previous_cost"`
}

type LLMCostTile struct {
	db *clickhouse.ClickhouseDB
}

func NewLLMCostTile(db *clickhouse.ClickhouseDB) *LLMCostTile {
	return &LLMCostTile{db: db}
}

func (t *LLMCostTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*LLMCostResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *LLMCostTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) (*LLMCostResponse, error) {
	query := t.buildQuery(filter)

	row := t.db.Db().QueryRow(ctx, query, filter.ProjectID)

	var data LLMCostResponse
	err := row.ScanStruct(&data)
	if err != nil {
		return nil, t.wrapFetchError(err, filter.ProjectID)
	}

	return &data, nil
}

func (t *LLMCostTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    sumIf(total_cost, start_time > now() - %[1]s) AS cost,
		    sumIf(total_cost, start_time BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_cost
		FROM llm_generations
		WHERE project_id = ?
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}

func (t *LLMCostTile) wrapFetchError(err error, projectID string) error {
	return errors.WithTelemetry(
		errors.Wrap(err, NewLLMCostFetchError().Error()),
		"project_id", projectID,
	)
}
