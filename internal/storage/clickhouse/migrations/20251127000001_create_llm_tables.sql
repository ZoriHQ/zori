-- +goose Up

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS llm_traces (
    project_id String,
    trace_id String,
    name Nullable(String),
    user_id Nullable(String),
    session_id Nullable(String),
    release Nullable(String),
    version Nullable(String),
    input String DEFAULT '{}',
    output String DEFAULT '{}',
    metadata String DEFAULT '{}',
    tags Array(String) DEFAULT [],
    public Bool DEFAULT false,
    timestamp DateTime64(3),
    created_at DateTime64(3) DEFAULT now64(3),
    updated_at DateTime64(3) DEFAULT now64(3),
    _version UInt64 DEFAULT 1
) ENGINE = ReplacingMergeTree(_version)
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, trace_id)
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS llm_generations (
    project_id String,
    generation_id String,
    trace_id String,
    parent_observation_id Nullable(String),
    name Nullable(String),
    model Nullable(String),
    model_parameters String DEFAULT '{}',
    input String DEFAULT '{}',
    output String DEFAULT '{}',
    start_time DateTime64(3),
    end_time Nullable(DateTime64(3)),
    completion_start_time Nullable(DateTime64(3)),
    latency_ms Nullable(UInt64),
    time_to_first_token_ms Nullable(UInt64),
    input_tokens Nullable(UInt32),
    output_tokens Nullable(UInt32),
    total_tokens Nullable(UInt32),
    input_cost Nullable(Float64),
    output_cost Nullable(Float64),
    total_cost Nullable(Float64),
    level LowCardinality(String) DEFAULT 'DEFAULT',
    status_message Nullable(String),
    metadata String DEFAULT '{}',
    prompt_name Nullable(String),
    prompt_version Nullable(UInt32),
    created_at DateTime64(3) DEFAULT now64(3),
    updated_at DateTime64(3) DEFAULT now64(3),
    _version UInt64 DEFAULT 1
) ENGINE = ReplacingMergeTree(_version)
PARTITION BY toYYYYMM(start_time)
ORDER BY (project_id, start_time, trace_id, generation_id)
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS llm_generations;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS llm_traces;
-- +goose StatementEnd
