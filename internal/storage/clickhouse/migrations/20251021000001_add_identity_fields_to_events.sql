-- +goose Up
-- +goose StatementBegin
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS user_id Nullable(String) COMMENT 'Authenticated user ID when visitor is identified',
    ADD COLUMN IF NOT EXISTS external_id Nullable(String) COMMENT 'External/customer-provided ID for visitor identification',
    ADD COLUMN IF NOT EXISTS email_hash Nullable(FixedString(64)) COMMENT 'SHA256 hash of email for privacy-safe analytics';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events (user_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_events_external_id ON events (external_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_events_email_hash ON events (email_hash) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_events_email_hash ON events;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_events_external_id ON events;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_events_user_id ON events;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events
    DROP COLUMN IF EXISTS email_hash,
    DROP COLUMN IF EXISTS external_id,
    DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
