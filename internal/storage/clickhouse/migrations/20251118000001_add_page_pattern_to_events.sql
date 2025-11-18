-- +goose Up
-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS page_pattern Nullable(String);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_page_pattern ON events (page_pattern) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_page_pattern;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS page_pattern;
-- +goose StatementEnd
