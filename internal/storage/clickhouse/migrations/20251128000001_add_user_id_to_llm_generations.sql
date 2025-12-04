-- +goose Up
-- +goose StatementBegin
ALTER TABLE llm_generations ADD COLUMN user_id Nullable(String) AFTER trace_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE llm_generations DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
