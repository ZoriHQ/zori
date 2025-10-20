-- +goose Up
-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS click_on;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS click_on Nullable(String);
-- +goose StatementEnd
