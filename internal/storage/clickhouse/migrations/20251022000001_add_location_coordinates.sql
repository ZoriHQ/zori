-- +goose Up
-- +goose StatementBegin
-- Add latitude and longitude columns to events table
ALTER TABLE events
ADD COLUMN IF NOT EXISTS location_latitude Nullable(Float64),
ADD COLUMN IF NOT EXISTS location_longitude Nullable(Float64);
-- +goose StatementEnd

-- +goose StatementBegin
-- Add latitude and longitude columns to sessions table
ALTER TABLE sessions
ADD COLUMN IF NOT EXISTS location_latitude SimpleAggregateFunction(any, Nullable(Float64)),
ADD COLUMN IF NOT EXISTS location_longitude SimpleAggregateFunction(any, Nullable(Float64));
-- +goose StatementEnd

-- +goose StatementBegin
-- Drop the existing materialized view
DROP VIEW IF EXISTS sessions_mv;
-- +goose StatementEnd

-- +goose StatementBegin
-- Recreate the materialized view with latitude and longitude fields
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
    anySimpleState(location_latitude) as location_latitude,
    anySimpleState(location_longitude) as location_longitude,
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
-- Drop the materialized view
DROP VIEW IF EXISTS sessions_mv;
-- +goose StatementEnd

-- +goose StatementBegin
-- Recreate the materialized view without latitude and longitude fields
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

-- +goose StatementBegin
-- Remove latitude and longitude columns from sessions table
ALTER TABLE sessions
DROP COLUMN IF EXISTS location_latitude,
DROP COLUMN IF EXISTS location_longitude;
-- +goose StatementEnd

-- +goose StatementBegin
-- Remove latitude and longitude columns from events table
ALTER TABLE events
DROP COLUMN IF EXISTS location_latitude,
DROP COLUMN IF EXISTS location_longitude;
-- +goose StatementEnd
