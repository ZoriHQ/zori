-- +goose Up
-- +goose StatementBegin
-- +goose StatementEnd
ALTER TABLE projects DROP COLUMN langfuse_secret_key_hash;
-- +goose Down
-- +goose StatementBegin
ALTER TABLE projects ADD COLUMN langfuse_secret_key_hash VARCHAR(255);
-- +goose StatementEnd
