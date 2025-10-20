-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS payment_events (
    -- Payment identification
    payment_id String,
    visitor_id Nullable(String),  -- Nullable for unattributed payments from historical sync

    -- Payment details
    provider_type LowCardinality(String),
    payment_status LowCardinality(String),  -- succeeded, failed, pending, refunded
    product_name String,

    -- Amounts (stored in smallest currency unit, e.g., cents)
    amount Int64,
    currency FixedString(3),  -- ISO 4217 currency code (USD, EUR, etc.)

    -- Timestamps
    payment_timestamp_utc DateTime64(3, 'UTC'),  -- When payment actually occurred
    server_timestamp_utc DateTime64(3, 'UTC') DEFAULT now64(3, 'UTC'),

    -- Organization hierarchy
    project_id UUID,
    organization_id UUID,

    -- Additional metadata (stored as Map for flexibility)
    metadata Map(String, String),

    -- Metadata
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(payment_timestamp_utc)
ORDER BY (organization_id, project_id, payment_timestamp_utc, payment_id)
TTL payment_timestamp_utc + INTERVAL 3 YEAR
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_payment_visitor_id ON payment_events (visitor_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_payment_provider ON payment_events (provider_type) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_payment_status ON payment_events (payment_status) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payment_events;
-- +goose StatementEnd
