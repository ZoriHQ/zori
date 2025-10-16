-- +goose Up
-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS host String DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS host;
-- +goose StatementEnd
