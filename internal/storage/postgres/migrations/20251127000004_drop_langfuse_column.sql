-- +goose Up
-- +goose StatementBegin
ALTER TABLE projects DROP COLUMN IF EXISTS langfuse_secret_key_hash;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
ALTER TABLE projects ADD COLUMN IF NOT EXISTS langfuse_secret_key_hash VARCHAR(255);
-- +goose StatementEnd
