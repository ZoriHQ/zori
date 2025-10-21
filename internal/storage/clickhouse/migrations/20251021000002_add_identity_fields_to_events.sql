-- +goose Up
-- +goose StatementBegin
ALTER TABLE events
  ADD COLUMN IF NOT EXISTS user_id Nullable(String),
  ADD COLUMN IF NOT EXISTS external_id Nullable(String),
  ADD COLUMN IF NOT EXISTS email_hash Nullable(FixedString(64));
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_user_id ON events (user_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_external_id ON events (external_id) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_external_id ON events;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_id ON events;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events
  DROP COLUMN IF EXISTS email_hash,
  DROP COLUMN IF EXISTS external_id,
  DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
