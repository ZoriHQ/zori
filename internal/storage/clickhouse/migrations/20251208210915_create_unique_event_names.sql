-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS unique_event_names (
    organization_id String,
    project_id String,
    event_name String,
    first_seen SimpleAggregateFunction(min, DateTime64(3, 'UTC')),
    last_seen SimpleAggregateFunction(max, DateTime64(3, 'UTC')),
    event_count SimpleAggregateFunction(sum, UInt64)
) ENGINE = AggregatingMergeTree()
ORDER BY (organization_id, project_id, event_name)
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO unique_event_names
SELECT
    organization_id,
    project_id,
    event_name,
    minSimpleState(client_timestamp_utc) as first_seen,
    maxSimpleState(client_timestamp_utc) as last_seen,
    sumSimpleState(toUInt64(1)) as event_count
FROM events
WHERE event_name IS NOT NULL AND event_name != ''
GROUP BY organization_id, project_id, event_name;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS unique_event_names_mv
TO unique_event_names
AS SELECT
    organization_id,
    project_id,
    event_name,
    minSimpleState(client_timestamp_utc) as first_seen,
    maxSimpleState(client_timestamp_utc) as last_seen,
    sumSimpleState(toUInt64(1)) as event_count
FROM events
WHERE event_name IS NOT NULL AND event_name != ''
GROUP BY organization_id, project_id, event_name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS unique_event_names_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS unique_event_names;
-- +goose StatementEnd
