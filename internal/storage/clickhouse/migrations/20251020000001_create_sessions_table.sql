-- +goose Up
-- +goose StatementBegin
-- Create the main sessions aggregation table
CREATE TABLE IF NOT EXISTS sessions (
    -- Session identification
    session_id String,
    visitor_id String,

    -- Organization hierarchy
    project_id UUID,
    organization_id UUID,

    -- Session timing
    started_at SimpleAggregateFunction(min, DateTime64(3, 'UTC')),
    ended_at SimpleAggregateFunction(max, DateTime64(3, 'UTC')),

    -- Session metrics
    page_count SimpleAggregateFunction(sum, UInt64),

    -- Entry and exit pages
    entry_page SimpleAggregateFunction(any, String),
    exit_page SimpleAggregateFunction(anyLast, String),

    -- UTM parameters (from first event in session)
    utm_source SimpleAggregateFunction(any, String),
    utm_medium SimpleAggregateFunction(any, String),
    utm_campaign SimpleAggregateFunction(any, String),

    -- Location (from first event)
    location_country_iso SimpleAggregateFunction(any, Nullable(String)),
    location_city SimpleAggregateFunction(any, Nullable(String)),

    -- Device info (from first event)
    device_type SimpleAggregateFunction(any, Nullable(String)),
    browser_name SimpleAggregateFunction(any, Nullable(String)),

    -- Metadata
    created_at DateTime DEFAULT now()
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(started_at)
ORDER BY (organization_id, project_id, session_id)
TTL started_at + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_sessions_visitor_id ON sessions (visitor_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions (started_at) TYPE minmax GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
-- Create materialized view to auto-populate sessions from events
CREATE MATERIALIZED VIEW IF NOT EXISTS sessions_mv TO sessions
AS
SELECT
    session_id,
    visitor_id,
    project_id,
    organization_id,
    minSimpleState(client_timestamp_utc) as started_at,
    maxSimpleState(client_timestamp_utc) as ended_at,
    sumSimpleState(toUInt32(1)) as page_count,
    anySimpleState(page_url) as entry_page,
    anyLastSimpleState(page_url) as exit_page,
    anySimpleState(utm_parameters['utm_source']) as utm_source,
    anySimpleState(utm_parameters['utm_medium']) as utm_medium,
    anySimpleState(utm_parameters['utm_campaign']) as utm_campaign,
    anySimpleState(location_country_iso) as location_country_iso,
    anySimpleState(location_city) as location_city,
    anySimpleState(device_type) as device_type,
    anySimpleState(browser_name) as browser_name
FROM events
WHERE session_id != ''
GROUP BY
    session_id,
    visitor_id,
    project_id,
    organization_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS sessions_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_sessions_started_at ON sessions;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_sessions_visitor_id ON sessions;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd
