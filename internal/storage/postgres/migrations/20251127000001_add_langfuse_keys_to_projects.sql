-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE projects ADD COLUMN langfuse_public_key VARCHAR(50) UNIQUE;
ALTER TABLE projects ADD COLUMN langfuse_secret_key VARCHAR(60);

CREATE INDEX idx_projects_langfuse_public_key ON projects(langfuse_public_key) WHERE langfuse_public_key IS NOT NULL;

-- Generate keys for existing projects
UPDATE projects
SET
    langfuse_public_key = 'pk-lf-' || encode(gen_random_bytes(16), 'hex'),
    langfuse_secret_key = 'sk-lf-' || encode(gen_random_bytes(24), 'hex')
WHERE langfuse_public_key IS NULL;

-- Create trigger function to auto-generate langfuse keys on INSERT
CREATE OR REPLACE FUNCTION generate_langfuse_keys()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.langfuse_public_key IS NULL THEN
        NEW.langfuse_public_key := 'pk-lf-' || encode(gen_random_bytes(16), 'hex');
    END IF;
    IF NEW.langfuse_secret_key IS NULL THEN
        NEW.langfuse_secret_key := 'sk-lf-' || encode(gen_random_bytes(24), 'hex');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to projects table
CREATE TRIGGER trigger_generate_langfuse_keys
    BEFORE INSERT ON projects
    FOR EACH ROW
    EXECUTE FUNCTION generate_langfuse_keys();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trigger_generate_langfuse_keys ON projects;
DROP FUNCTION IF EXISTS generate_langfuse_keys();
DROP INDEX IF EXISTS idx_projects_langfuse_public_key;
ALTER TABLE projects DROP COLUMN IF EXISTS langfuse_secret_key;
ALTER TABLE projects DROP COLUMN IF EXISTS langfuse_public_key;
-- +goose StatementEnd
