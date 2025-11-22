-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS llm_traces (
    trace_id UUID DEFAULT generateUUIDv4(),

    visitor_id Nullable(String),
    customer_id Nullable(String),

    provider String,
    model String,

    input_tokens UInt32,
    output_tokens UInt32,

    input_prompt Nullable(String),
    output_prompt Nullable(String),

    input_cost Decimal(18, 10),
    output_cost Decimal(18, 10),
    extra_fee Nullable(Decimal(18, 10)),
    total_cost Decimal(18, 10),

    project_id String,
    organization_id String,

    client_timestamp_utc DateTime64(3, 'UTC'),
    server_timestamp_utc DateTime64(3, 'UTC') DEFAULT now64(3, 'UTC'),

    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(client_timestamp_utc)
ORDER BY (organization_id, project_id, client_timestamp_utc, trace_id)
TTL client_timestamp_utc + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_llm_traces_visitor_id ON llm_traces (visitor_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_llm_traces_customer_id ON llm_traces (customer_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_llm_traces_provider ON llm_traces (provider) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_llm_traces_model ON llm_traces (model) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS llm_traces_daily_costs_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (organization_id, project_id, date, provider, model)
AS SELECT
    organization_id,
    project_id,
    toDate(client_timestamp_utc) AS date,
    provider,
    model,
    sum(input_tokens) AS total_input_tokens,
    sum(output_tokens) AS total_output_tokens,
    sum(total_cost) AS total_cost,
    count() AS trace_count
FROM llm_traces
GROUP BY organization_id, project_id, date, provider, model;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS llm_traces_customer_costs_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (organization_id, project_id, date, customer_id)
SETTINGS allow_nullable_key = 1
AS SELECT
    organization_id,
    project_id,
    toDate(client_timestamp_utc) AS date,
    customer_id,
    sum(input_tokens) AS total_input_tokens,
    sum(output_tokens) AS total_output_tokens,
    sum(total_cost) AS total_cost,
    count() AS trace_count
FROM llm_traces
WHERE customer_id IS NOT NULL
GROUP BY organization_id, project_id, date, customer_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS llm_traces_customer_costs_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP VIEW IF EXISTS llm_traces_daily_costs_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS llm_traces;
-- +goose StatementEnd
