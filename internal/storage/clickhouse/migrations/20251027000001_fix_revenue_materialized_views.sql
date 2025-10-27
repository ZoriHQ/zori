-- +goose Up
-- +goose StatementBegin
-- Drop existing materialized views that have the counting issue
DROP VIEW IF EXISTS revenue_timeline_hourly_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP VIEW IF EXISTS revenue_timeline_daily_mv;
-- +goose StatementEnd

-- +goose StatementBegin
-- Recreate hourly materialized view with proper DISTINCT counting
-- This fixes the issue where duplicate payment rows were being counted multiple times
CREATE MATERIALIZED VIEW IF NOT EXISTS revenue_timeline_hourly_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(time_bucket)
ORDER BY (organization_id, project_id, time_bucket)
POPULATE
AS SELECT
    organization_id,
    project_id,
    toStartOfHour(payment_timestamp_utc) as time_bucket,
    currency,
    sum(amount) as total_revenue,
    count(*) as payment_count
FROM (
    SELECT DISTINCT
        organization_id,
        project_id,
        payment_id,
        payment_timestamp_utc,
        currency,
        amount
    FROM payment_events
    WHERE payment_status = 'succeeded'
)
GROUP BY
    organization_id,
    project_id,
    toStartOfHour(payment_timestamp_utc),
    currency;
-- +goose StatementEnd

-- +goose StatementBegin
-- Recreate daily materialized view with proper DISTINCT counting
CREATE MATERIALIZED VIEW IF NOT EXISTS revenue_timeline_daily_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(time_bucket)
ORDER BY (organization_id, project_id, time_bucket)
POPULATE
AS SELECT
    organization_id,
    project_id,
    toDate(payment_timestamp_utc) as time_bucket,
    currency,
    sum(amount) as total_revenue,
    count(*) as payment_count
FROM (
    SELECT DISTINCT
        organization_id,
        project_id,
        payment_id,
        payment_timestamp_utc,
        currency,
        amount
    FROM payment_events
    WHERE payment_status = 'succeeded'
)
GROUP BY
    organization_id,
    project_id,
    toDate(payment_timestamp_utc),
    currency;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS revenue_timeline_daily_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP VIEW IF EXISTS revenue_timeline_hourly_mv;
-- +goose StatementEnd

-- +goose StatementBegin
-- Restore original hourly materialized view (with the bug)
CREATE MATERIALIZED VIEW IF NOT EXISTS revenue_timeline_hourly_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(time_bucket)
ORDER BY (organization_id, project_id, time_bucket)
POPULATE
AS SELECT
    organization_id,
    project_id,
    toStartOfHour(payment_timestamp_utc) as time_bucket,
    currency,
    sum(amount) as total_revenue,
    count(*) as payment_count
FROM payment_events
WHERE payment_status = 'succeeded'
GROUP BY
    organization_id,
    project_id,
    toStartOfHour(payment_timestamp_utc),
    currency;
-- +goose StatementEnd

-- +goose StatementBegin
-- Restore original daily materialized view (with the bug)
CREATE MATERIALIZED VIEW IF NOT EXISTS revenue_timeline_daily_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(time_bucket)
ORDER BY (organization_id, project_id, time_bucket)
POPULATE
AS SELECT
    organization_id,
    project_id,
    toDate(payment_timestamp_utc) as time_bucket,
    currency,
    sum(amount) as total_revenue,
    count(*) as payment_count
FROM payment_events
WHERE payment_status = 'succeeded'
GROUP BY
    organization_id,
    project_id,
    toDate(payment_timestamp_utc),
    currency;
-- +goose StatementEnd
