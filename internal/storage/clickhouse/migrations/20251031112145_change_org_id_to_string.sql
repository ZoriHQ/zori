-- +goose Up
-- Change organization_id and project_id from UUID to String
-- Strategy: Add new columns, copy data, drop old columns, rename new columns

-- ====================================
-- EVENTS TABLE
-- ====================================

-- Add new String columns
-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS organization_id_new String;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS project_id_new String;
-- +goose StatementEnd

-- Copy data from UUID to String columns
-- +goose StatementBegin
ALTER TABLE events UPDATE organization_id_new = toString(organization_id) WHERE organization_id_new = '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events UPDATE project_id_new = toString(project_id) WHERE project_id_new = '';
-- +goose StatementEnd

-- Drop dependent materialized views first
-- +goose StatementBegin
DROP VIEW IF EXISTS sessions_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP VIEW IF EXISTS revenue_by_project_daily;
-- +goose StatementEnd

-- Create new events table with String types
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS events_new (
    event_name Nullable(String),
    client_generated_event_id UUID,
    visitor_id String,
    session_id String,
    user_id Nullable(String),
    external_id Nullable(String),
    email_hash Nullable(String),
    client_timestamp_utc DateTime64(3, 'UTC'),
    server_timestamp_utc DateTime64(3, 'UTC') DEFAULT now64(3, 'UTC'),
    user_agent String,
    ip String,
    referrer_url String,
    referrer_domain Nullable(String),
    referrer_path Nullable(String),
    page_url String,
    page_path String,
    host String,
    browser_name Nullable(String),
    os_name Nullable(String),
    device_type Nullable(String),
    click_element_tag Nullable(String),
    click_element_selector Nullable(String),
    click_element_text Nullable(String),
    click_position_x Nullable(Float64),
    click_position_y Nullable(Float64),
    click_screen_width Nullable(UInt16),
    click_screen_height Nullable(UInt16),
    click_element_type Nullable(String),
    click_element_category Nullable(String),
    is_cta_click Nullable(Bool),
    link_destination Nullable(String),
    is_external_link Nullable(Bool),
    is_download_link Nullable(Bool),
    utm_parameters Map(String, String),
    custom_properties String,
    project_id String,
    organization_id String,
    location_country_iso Nullable(FixedString(2)),
    location_city Nullable(String),
    location_latitude Nullable(Float64),
    location_longitude Nullable(Float64),
    created_at DateTime DEFAULT now(),
    utm_source String MATERIALIZED utm_parameters['utm_source'],
    utm_medium String MATERIALIZED utm_parameters['utm_medium'],
    utm_campaign String MATERIALIZED utm_parameters['utm_campaign']
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(client_timestamp_utc)
ORDER BY (organization_id, project_id, client_timestamp_utc, visitor_id)
TTL client_timestamp_utc + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- Copy data with UUID to String conversion
-- +goose StatementBegin
INSERT INTO events_new SELECT
    event_name,
    client_generated_event_id,
    visitor_id,
    session_id,
    user_id,
    external_id,
    email_hash,
    client_timestamp_utc,
    server_timestamp_utc,
    user_agent,
    ip,
    referrer_url,
    referrer_domain,
    referrer_path,
    page_url,
    page_path,
    host,
    browser_name,
    os_name,
    device_type,
    click_element_tag,
    click_element_selector,
    click_element_text,
    click_position_x,
    click_position_y,
    click_screen_width,
    click_screen_height,
    click_element_type,
    click_element_category,
    is_cta_click,
    link_destination,
    is_external_link,
    is_download_link,
    utm_parameters,
    custom_properties,
    toString(project_id),
    toString(organization_id),
    location_country_iso,
    location_city,
    location_latitude,
    location_longitude,
    created_at
FROM events;
-- +goose StatementEnd

-- Drop old table and rename new table
-- +goose StatementBegin
DROP TABLE events;
-- +goose StatementEnd

-- +goose StatementBegin
RENAME TABLE events_new TO events;
-- +goose StatementEnd

-- Recreate indexes
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_visitor_id ON events (visitor_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_event_name ON events (event_name) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_utm_source ON events (utm_source) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_utm_campaign ON events (utm_campaign) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- ====================================
-- PAYMENT_EVENTS TABLE
-- ====================================

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS payment_events_new (
    payment_id String,
    visitor_id Nullable(String),
    provider_type LowCardinality(String),
    payment_status LowCardinality(String),
    product_name String,
    amount Int64,
    currency FixedString(3),
    payment_timestamp_utc DateTime64(3, 'UTC'),
    server_timestamp_utc DateTime64(3, 'UTC') DEFAULT now64(3, 'UTC'),
    project_id String,
    organization_id String,
    metadata Map(String, String),
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(payment_timestamp_utc)
ORDER BY (organization_id, project_id, payment_timestamp_utc, payment_id)
TTL payment_timestamp_utc + INTERVAL 3 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO payment_events_new SELECT
    payment_id,
    visitor_id,
    provider_type,
    payment_status,
    product_name,
    amount,
    currency,
    payment_timestamp_utc,
    server_timestamp_utc,
    toString(project_id),
    toString(organization_id),
    metadata,
    created_at
FROM payment_events;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE payment_events;
-- +goose StatementEnd

-- +goose StatementBegin
RENAME TABLE payment_events_new TO payment_events;
-- +goose StatementEnd

-- Recreate indexes
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_payment_visitor_id ON payment_events (visitor_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_payment_provider ON payment_events (provider_type) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_payment_status ON payment_events (payment_status) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- ====================================
-- SESSIONS TABLE
-- ====================================

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS sessions_new (
    session_id String,
    visitor_id String,
    project_id String,
    organization_id String,
    started_at SimpleAggregateFunction(min, DateTime64(3, 'UTC')),
    ended_at SimpleAggregateFunction(max, DateTime64(3, 'UTC')),
    page_count SimpleAggregateFunction(sum, UInt64),
    entry_page SimpleAggregateFunction(any, String),
    exit_page SimpleAggregateFunction(anyLast, String),
    utm_source SimpleAggregateFunction(any, String),
    utm_medium SimpleAggregateFunction(any, String),
    utm_campaign SimpleAggregateFunction(any, String),
    location_country_iso SimpleAggregateFunction(any, Nullable(String)),
    location_city SimpleAggregateFunction(any, Nullable(String)),
    device_type SimpleAggregateFunction(any, Nullable(String)),
    browser_name SimpleAggregateFunction(any, Nullable(String)),
    created_at DateTime DEFAULT now()
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(started_at)
ORDER BY (organization_id, project_id, session_id)
TTL started_at + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- Note: Sessions table is populated by materialized view, we'll let it repopulate
-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd

-- +goose StatementBegin
RENAME TABLE sessions_new TO sessions;
-- +goose StatementEnd

-- Recreate indexes
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_sessions_visitor_id ON sessions (visitor_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions (started_at) TYPE minmax GRANULARITY 1;
-- +goose StatementEnd

-- Recreate sessions materialized view
-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS sessions_mv TO sessions
AS
SELECT
    session_id,
    visitor_id,
    project_id,
    organization_id,
    minSimpleState(client_timestamp_utc) as started_at,
    maxSimpleState(client_timestamp_utc) as ended_at,
    sumSimpleState(toUInt64(1)) as page_count,
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

-- ====================================
-- VISITOR_SUMMARY TABLE
-- ====================================

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS visitor_summary_new (
    visitor_id String,
    project_id String,
    organization_id String,
    user_id Nullable(String),
    external_id Nullable(String),
    email_hash Nullable(String),
    first_seen DateTime64(3, 'UTC'),
    last_seen DateTime64(3, 'UTC'),
    total_sessions UInt32,
    total_page_views UInt32,
    total_events UInt32,
    first_referrer_url String,
    first_referrer_domain Nullable(String),
    first_utm_source Nullable(String),
    first_utm_medium Nullable(String),
    first_utm_campaign Nullable(String),
    latest_device_type Nullable(String),
    latest_browser_name Nullable(String),
    latest_os_name Nullable(String),
    latest_location_country_iso Nullable(FixedString(2)),
    latest_location_city Nullable(String),
    created_at DateTime DEFAULT now(),
    updated_at DateTime DEFAULT now()
) ENGINE = ReplacingMergeTree(updated_at)
PARTITION BY toYYYYMM(first_seen)
ORDER BY (organization_id, project_id, visitor_id)
TTL first_seen + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO visitor_summary_new SELECT
    visitor_id,
    toString(project_id),
    toString(organization_id),
    NULL as user_id,
    NULL as external_id,
    NULL as email_hash,
    first_seen,
    last_seen,
    total_sessions,
    0 as total_page_views,
    total_events,
    '' as first_referrer_url,
    NULL as first_referrer_domain,
    NULL as first_utm_source,
    NULL as first_utm_medium,
    NULL as first_utm_campaign,
    device_type as latest_device_type,
    browser_name as latest_browser_name,
    NULL as latest_os_name,
    location_country_iso as latest_location_country_iso,
    location_city as latest_location_city,
    created_at,
    now() as updated_at
FROM visitor_summary;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE visitor_summary;
-- +goose StatementEnd

-- +goose StatementBegin
RENAME TABLE visitor_summary_new TO visitor_summary;
-- +goose StatementEnd

-- ====================================
-- RECREATE REVENUE MATERIALIZED VIEW
-- ====================================

-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS revenue_by_project_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (organization_id, project_id, date, provider_type, currency)
POPULATE AS
SELECT
    organization_id,
    project_id,
    toDate(payment_timestamp_utc) as date,
    provider_type,
    currency,
    sum(amount) as total_amount,
    count() as payment_count,
    countIf(payment_status = 'succeeded') as successful_payments,
    countIf(payment_status = 'failed') as failed_payments,
    countIf(payment_status = 'refunded') as refunded_payments,
    sumIf(amount, payment_status = 'succeeded') as successful_amount,
    sumIf(amount, payment_status = 'refunded') as refunded_amount
FROM payment_events
GROUP BY organization_id, project_id, date, provider_type, currency;
-- +goose StatementEnd

-- +goose Down
-- Rollback would require converting String back to UUID
-- +goose StatementBegin
SELECT 'Migration rollback not supported - would need UUID conversion';
-- +goose StatementEnd
