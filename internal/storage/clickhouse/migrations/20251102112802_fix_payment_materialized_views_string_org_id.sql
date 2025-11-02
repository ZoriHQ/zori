-- +goose Up
-- +goose StatementBegin
-- Drop existing materialized views that use UUID types
DROP VIEW IF EXISTS revenue_timeline_hourly_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP VIEW IF EXISTS revenue_timeline_daily_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP VIEW IF EXISTS visitor_first_touch_attribution_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS visitor_first_touch_attribution;
-- +goose StatementEnd

-- +goose StatementBegin
-- Recreate visitor first-touch attribution table with String types
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

-- +goose StatementBegin
-- Populate the first-touch attribution table from existing events
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

-- +goose StatementBegin
-- Recreate materialized view to keep first-touch attribution updated
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

-- +goose StatementBegin
-- Recreate hourly materialized view with String types and proper DISTINCT counting
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
-- Recreate daily materialized view with String types and proper DISTINCT counting
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
DROP VIEW IF EXISTS visitor_first_touch_attribution_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS visitor_first_touch_attribution;
-- +goose StatementEnd

-- +goose StatementBegin
-- Restore with UUID types (from original migration)
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

-- +goose StatementBegin
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
