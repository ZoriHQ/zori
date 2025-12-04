-- +goose Up

-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS llm_daily_costs_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, model)
AS SELECT
    project_id,
    toDate(start_time) AS date,
    coalesce(model, 'unknown') AS model,
    count() AS generation_count,
    sum(input_tokens) AS total_input_tokens,
    sum(output_tokens) AS total_output_tokens,
    sum(total_tokens) AS total_tokens,
    sum(total_cost) AS total_cost,
    avg(latency_ms) AS avg_latency_ms
FROM llm_generations
GROUP BY project_id, date, model;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS llm_hourly_throughput_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (project_id, hour)
AS SELECT
    project_id,
    toStartOfHour(start_time) AS hour,
    count() AS generation_count,
    countIf(level = 'ERROR') AS error_count,
    sum(total_tokens) AS total_tokens,
    sum(total_cost) AS total_cost
FROM llm_generations
GROUP BY project_id, hour;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP VIEW IF EXISTS llm_hourly_throughput_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP VIEW IF EXISTS llm_daily_costs_mv;
-- +goose StatementEnd
