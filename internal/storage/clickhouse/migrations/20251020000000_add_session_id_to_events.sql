-- +goose Up
-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS session_id String DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_session_id ON events (session_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_session_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS session_id;
-- +goose StatementEnd
