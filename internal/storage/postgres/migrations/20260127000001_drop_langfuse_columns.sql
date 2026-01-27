-- +goose Up
-- Drop langfuse trigger and function first
DROP TRIGGER IF EXISTS trigger_generate_langfuse_keys ON projects;
DROP FUNCTION IF EXISTS generate_langfuse_keys();
DROP INDEX IF EXISTS idx_projects_langfuse_public_key;

-- Drop langfuse columns
ALTER TABLE projects DROP COLUMN IF EXISTS langfuse_public_key;
ALTER TABLE projects DROP COLUMN IF EXISTS langfuse_secret_key;

-- +goose Down
-- Data loss expected on rollback - columns and trigger would need to be recreated manually
