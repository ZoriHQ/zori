-- +goose Up
-- Remove refresh_token column and its index from sessions table
ALTER TABLE sessions DROP COLUMN IF EXISTS refresh_token;

-- Add updated_at column to sessions table
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Create trigger to update updated_at timestamp for sessions
CREATE TRIGGER update_sessions_updated_at BEFORE UPDATE ON sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
-- Remove the trigger
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;

-- Remove updated_at column
ALTER TABLE sessions DROP COLUMN IF EXISTS updated_at;

-- Add back refresh_token column with unique constraint
ALTER TABLE sessions ADD COLUMN refresh_token VARCHAR(500) UNIQUE NOT NULL DEFAULT '';

-- Re-create the index for refresh_token
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
