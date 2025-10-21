-- +goose Up
-- +goose StatementBegin
-- Create the visitors identity table to store identified visitor information
CREATE TABLE IF NOT EXISTS visitors (
    -- Primary identity
    visitor_id VARCHAR(255) PRIMARY KEY,

    -- Organization hierarchy
    project_id UUID NOT NULL,
    organization_id UUID NOT NULL,

    -- User identity fields (nullable - only set when visitor is identified)
    user_id VARCHAR(255),
    external_id VARCHAR(255),
    email VARCHAR(255),
    email_hash VARCHAR(64), -- SHA256 hash for privacy-safe matching

    -- Optional profile fields
    name VARCHAR(255),
    phone VARCHAR(50),

    -- Metadata stored as JSONB for flexibility
    custom_traits JSONB DEFAULT '{}'::JSONB,

    -- Timestamps
    first_identified_at TIMESTAMP WITH TIME ZONE,
    last_identified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Index for looking up by user_id
CREATE INDEX IF NOT EXISTS idx_visitors_user_id ON visitors(user_id) WHERE user_id IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Index for looking up by external_id
CREATE INDEX IF NOT EXISTS idx_visitors_external_id ON visitors(external_id) WHERE external_id IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Index for looking up by email
CREATE INDEX IF NOT EXISTS idx_visitors_email ON visitors(email) WHERE email IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Index for looking up by email_hash (for privacy-safe joins)
CREATE INDEX IF NOT EXISTS idx_visitors_email_hash ON visitors(email_hash) WHERE email_hash IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Composite index for project-level queries
CREATE INDEX IF NOT EXISTS idx_visitors_project_org ON visitors(project_id, organization_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Updated_at trigger
CREATE OR REPLACE FUNCTION update_visitors_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trigger_update_visitors_updated_at
    BEFORE UPDATE ON visitors
    FOR EACH ROW
    EXECUTE FUNCTION update_visitors_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trigger_update_visitors_updated_at ON visitors;
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION IF EXISTS update_visitors_updated_at();
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitors_project_org;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitors_email_hash;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitors_email;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitors_external_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_visitors_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS visitors;
-- +goose StatementEnd
