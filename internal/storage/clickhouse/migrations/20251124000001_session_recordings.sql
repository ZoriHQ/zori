-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS session_recordings (
    id UUID DEFAULT generateUUIDv4(),

    visitor_id String,
    session_id String,

    project_id UUID,
    organization_id UUID,

    page_url String,
    host String,

    events String,
    event_count UInt32,

    chunk_index UInt32 DEFAULT 0,
    is_final_chunk UInt8 DEFAULT 0,

    client_timestamp_utc DateTime64(3, 'UTC'),
    server_timestamp_utc DateTime64(3, 'UTC') DEFAULT now64(3, 'UTC'),

    user_agent String,
    ip String,

    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(client_timestamp_utc)
ORDER BY (organization_id, project_id, session_id, client_timestamp_utc, chunk_index)
TTL client_timestamp_utc + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_sr_visitor_id ON session_recordings (visitor_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_sr_session_id ON session_recordings (session_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS session_recordings;
-- +goose StatementEnd
