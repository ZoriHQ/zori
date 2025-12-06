package tiles

import (
	"fmt"
	"zori/internal/ctx"
	"zori/internal/storage/clickhouse"
	"zori/services/analytics/filters"

	"github.com/cockroachdb/errors"
)

type LLMModelCostData struct {
	Model        string  `json:"model" ch:"model"`
	Cost         float64 `json:"cost" ch:"cost"`
	PreviousCost float64 `json:"previous_cost" ch:"previous_cost"`
}

type LLMTopModelsCostResponse struct {
	Data []*LLMModelCostData `json:"data"`
}

type LLMTopModelsCostTile struct {
	db *clickhouse.ClickhouseDB
}

func NewLLMTopModelsCostTile(db *clickhouse.ClickhouseDB) *LLMTopModelsCostTile {
	return &LLMTopModelsCostTile{db: db}
}

func (t *LLMTopModelsCostTile) Fetch(ctx *ctx.Ctx, filter *filters.SectionFilter) (*LLMTopModelsCostResponse, error) {
	data, err := t.fetchData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &LLMTopModelsCostResponse{Data: data}, nil
}

func (t *LLMTopModelsCostTile) fetchData(ctx *ctx.Ctx, filter *filters.SectionFilter) ([]*LLMModelCostData, error) {
	query := t.buildQuery(filter)

	rows, err := t.db.Db().Query(ctx, query, filter.ProjectID)
	if err != nil {
		return nil, t.wrapFetchError(err, filter.ProjectID)
	}
	defer clickhouse.EnsureClosed(rows)

	if rows.Err() != nil {
		return nil, t.wrapFetchError(rows.Err(), filter.ProjectID)
	}

	var models []*LLMModelCostData
	for rows.Next() {
		var data LLMModelCostData
		err := rows.ScanStruct(&data)
		if err != nil {
			return nil, t.wrapFetchError(err, filter.ProjectID)
		}
		models = append(models, &data)
	}

	return models, nil
}

func (t *LLMTopModelsCostTile) buildQuery(filter *filters.SectionFilter) string {
	return fmt.Sprintf(`
		SELECT
		    ifNull(nullIf(model, ''), 'unknown') AS model,
		    sumIf(total_cost, start_time > now() - %[1]s) AS cost,
		    sumIf(total_cost, start_time BETWEEN now() - %[2]s AND now() - %[1]s) AS previous_cost
		FROM llm_generations
		WHERE project_id = ?
		GROUP BY model
		HAVING cost > 0
		ORDER BY cost DESC
		LIMIT 3
	`, filter.TimeRange.IntervalValue, filter.TimeRange.IntervalValueDelta)
}

func (t *LLMTopModelsCostTile) wrapFetchError(err error, projectID string) error {
	return errors.WithTelemetry(
		errors.Wrap(err, NewLLMModelsCostFetchError().Error()),
		"project_id", projectID,
	)
}
