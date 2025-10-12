-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS events (
    -- Event identification
    event_name Nullable(String),
    client_generated_event_id UUID,
    visitor_id String,

    -- Timestamps
    client_timestamp_utc DateTime64(3, 'UTC'),
    server_timestamp_utc DateTime64(3, 'UTC') DEFAULT now64(3, 'UTC'),

    -- Request metadata
    user_agent String,
    ip String,
    referrer_url String,

    referrer_domain Nullable(String),
    referrer_path Nullable(String),

    page_url String,
    page_path String,

    browser_name Nullable(String),
    os_name Nullable(String),
    device_type Nullable(String),

    -- Interaction data
    click_on Nullable(String),
    click_position_x Nullable(Float64),
    click_position_y Nullable(Float64),

    -- UTM parameters (stored as Map for flexibility)
    utm_parameters Map(String, String),

    -- Custom properties (stored as JSON string for flexibility)
    custom_properties String,

    -- Organization hierarchy
    project_id UUID,
    organization_id UUID,

    -- Location
    location_country_iso Nullable(FixedString(2)),
    location_city Nullable(String),

    -- Metadata
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(client_timestamp_utc)
ORDER BY (organization_id, project_id, client_timestamp_utc, visitor_id)
TTL client_timestamp_utc + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_visitor_id ON events (visitor_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_event_name ON events (event_name) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS utm_source String MATERIALIZED utm_parameters['utm_source'];
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS utm_medium String MATERIALIZED utm_parameters['utm_medium'];
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS utm_campaign String MATERIALIZED utm_parameters['utm_campaign'];
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_utm_source ON events (utm_source) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_utm_campaign ON events (utm_campaign) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS events;
-- +goose StatementEnd
