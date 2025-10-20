-- +goose Up
-- +goose StatementBegin
-- Create the visitor summary aggregation table
CREATE TABLE IF NOT EXISTS visitor_summary (
    -- Visitor identification
    visitor_id String,

    -- Organization hierarchy
    project_id UUID,
    organization_id UUID,

    -- Visitor lifecycle
    first_seen SimpleAggregateFunction(min, DateTime64(3, 'UTC')),
    last_seen SimpleAggregateFunction(max, DateTime64(3, 'UTC')),

    -- Aggregate metrics
    total_sessions SimpleAggregateFunction(sum, UInt64),
    total_events SimpleAggregateFunction(sum, UInt64),

    -- Location (from most recent event)
    location_country_iso SimpleAggregateFunction(anyLast, Nullable(String)),
    location_city SimpleAggregateFunction(anyLast, Nullable(String)),

    -- Device info (from most recent event)
    device_type SimpleAggregateFunction(anyLast, Nullable(String)),
    browser_name SimpleAggregateFunction(anyLast, Nullable(String)),

    -- Metadata
    created_at DateTime DEFAULT now()
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(first_seen)
ORDER BY (organization_id, project_id, visitor_id)
TTL first_seen + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_visitor_summary_last_seen ON visitor_summary (last_seen) TYPE minmax GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_visitor_summary_first_seen ON visitor_summary (first_seen) TYPE minmax GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
-- Create materialized view to auto-populate visitor_summary from events
CREATE MATERIALIZED VIEW IF NOT EXISTS visitor_summary_mv TO visitor_summary
AS
SELECT
    visitor_id,
    project_id,
    organization_id,
    minSimpleState(client_timestamp_utc) as first_seen,
    maxSimpleState(client_timestamp_utc) as last_seen,
    sumSimpleState(toUInt64(session_id != '' ? 1 : 0)) as total_sessions,
    sumSimpleState(toUInt64(1)) as total_events,
    anyLastSimpleState(location_country_iso) as location_country_iso,
    anyLastSimpleState(location_city) as location_city,
    anyLastSimpleState(device_type) as device_type,
    anyLastSimpleState(browser_name) as browser_name
FROM events
WHERE visitor_id != ''
GROUP BY
    visitor_id,
    project_id,
    organization_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS visitor_summary_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitor_summary_first_seen ON visitor_summary;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitor_summary_last_seen ON visitor_summary;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS visitor_summary;
-- +goose StatementEnd
