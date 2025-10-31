-- +goose Up
-- Fix visitor_first_touch_attribution to use String types instead of UUID
-- This aligns with the migration 20251031112145 that changed events table to use String

-- Drop the existing materialized view
-- +goose StatementBegin
DROP VIEW IF EXISTS visitor_first_touch_attribution_mv;
-- +goose StatementEnd

-- Drop the existing table
-- +goose StatementBegin
DROP TABLE IF EXISTS visitor_first_touch_attribution;
-- +goose StatementEnd

-- Recreate table with String types for organization_id and project_id
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS visitor_first_touch_attribution (
    organization_id String,
    project_id String,
    visitor_id String,
    first_referrer_domain AggregateFunction(argMin, Nullable(String), DateTime64(3, 'UTC')),
    first_utm_source AggregateFunction(argMin, String, DateTime64(3, 'UTC')),
    first_utm_medium AggregateFunction(argMin, String, DateTime64(3, 'UTC')),
    first_utm_campaign AggregateFunction(argMin, String, DateTime64(3, 'UTC'))
) ENGINE = AggregatingMergeTree()
ORDER BY (organization_id, project_id, visitor_id);
-- +goose StatementEnd

-- Populate from existing events
-- +goose StatementBegin
INSERT INTO visitor_first_touch_attribution
SELECT
    organization_id,
    project_id,
    visitor_id,
    argMinState(referrer_domain, client_timestamp_utc) as first_referrer_domain,
    argMinState(utm_parameters['utm_source'], client_timestamp_utc) as first_utm_source,
    argMinState(utm_parameters['utm_medium'], client_timestamp_utc) as first_utm_medium,
    argMinState(utm_parameters['utm_campaign'], client_timestamp_utc) as first_utm_campaign
FROM events
GROUP BY organization_id, project_id, visitor_id;
-- +goose StatementEnd

-- Recreate materialized view with String types
-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS visitor_first_touch_attribution_mv
TO visitor_first_touch_attribution
AS SELECT
    organization_id,
    project_id,
    visitor_id,
    argMinState(referrer_domain, client_timestamp_utc) as first_referrer_domain,
    argMinState(utm_parameters['utm_source'], client_timestamp_utc) as first_utm_source,
    argMinState(utm_parameters['utm_medium'], client_timestamp_utc) as first_utm_medium,
    argMinState(utm_parameters['utm_campaign'], client_timestamp_utc) as first_utm_campaign
FROM events
GROUP BY organization_id, project_id, visitor_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS visitor_first_touch_attribution_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS visitor_first_touch_attribution;
-- +goose StatementEnd

-- +goose StatementBegin
-- Recreate with UUID types (original schema)
CREATE TABLE IF NOT EXISTS visitor_first_touch_attribution (
    organization_id UUID,
    project_id UUID,
    visitor_id String,
    first_referrer_domain AggregateFunction(argMin, Nullable(String), DateTime64(3, 'UTC')),
    first_utm_source AggregateFunction(argMin, String, DateTime64(3, 'UTC')),
    first_utm_medium AggregateFunction(argMin, String, DateTime64(3, 'UTC')),
    first_utm_campaign AggregateFunction(argMin, String, DateTime64(3, 'UTC'))
) ENGINE = AggregatingMergeTree()
ORDER BY (organization_id, project_id, visitor_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS visitor_first_touch_attribution_mv
TO visitor_first_touch_attribution
AS SELECT
    organization_id,
    project_id,
    visitor_id,
    argMinState(referrer_domain, client_timestamp_utc) as first_referrer_domain,
    argMinState(utm_parameters['utm_source'], client_timestamp_utc) as first_utm_source,
    argMinState(utm_parameters['utm_medium'], client_timestamp_utc) as first_utm_medium,
    argMinState(utm_parameters['utm_campaign'], client_timestamp_utc) as first_utm_campaign
FROM events
GROUP BY organization_id, project_id, visitor_id;
-- +goose StatementEnd
