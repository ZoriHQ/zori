-- +goose Up
-- +goose StatementBegin
ALTER TABLE visitor_summary
    ADD COLUMN IF NOT EXISTS user_id SimpleAggregateFunction(anyLast, Nullable(String)) COMMENT 'Most recent user_id for identified visitors',
    ADD COLUMN IF NOT EXISTS external_id SimpleAggregateFunction(anyLast, Nullable(String)) COMMENT 'Most recent external_id for identified visitors',
    ADD COLUMN IF NOT EXISTS email_hash SimpleAggregateFunction(anyLast, Nullable(String)) COMMENT 'Most recent email_hash for identified visitors';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_visitor_summary_user_id ON visitor_summary (user_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_visitor_summary_external_id ON visitor_summary (external_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
DROP VIEW IF EXISTS visitor_summary_mv;
-- +goose StatementEnd

-- +goose StatementBegin
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
    anyLastSimpleState(browser_name) as browser_name,
    anyLastSimpleState(user_id) as user_id,
    anyLastSimpleState(external_id) as external_id,
    anyLastSimpleState(email_hash) as email_hash
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

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitor_summary_external_id ON visitor_summary;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitor_summary_user_id ON visitor_summary;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE visitor_summary
    DROP COLUMN IF EXISTS email_hash,
    DROP COLUMN IF EXISTS external_id,
    DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
