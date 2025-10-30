-- +goose Up
-- Create system table for OSS deployment configuration
CREATE TABLE system (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    projects_limit INTEGER,
    admin_username VARCHAR(255),
    admin_password_hash VARCHAR(255),
    default_org_id UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER update_system_updated_at BEFORE UPDATE ON system
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert a single row that will be used for OSS configuration
-- (will be populated by the init process if OSS mode is enabled)
INSERT INTO system (id) VALUES (gen_random_uuid());

-- +goose Down
DROP TRIGGER IF EXISTS update_system_updated_at ON system;
DROP TABLE IF EXISTS system;
