-- +goose Up
-- Change organization_id columns from UUID to VARCHAR to support Clerk's string-based IDs

-- Projects table
ALTER TABLE projects
    ALTER COLUMN organization_id TYPE VARCHAR(255);

-- Payment providers table
ALTER TABLE payment_providers
    ALTER COLUMN organization_id TYPE VARCHAR(255);

-- Visitors table
ALTER TABLE visitors
    ALTER COLUMN organization_id TYPE VARCHAR(255);

-- System table - default_org_id
ALTER TABLE system
    ALTER COLUMN default_org_id TYPE VARCHAR(255);

-- +goose Down
-- Revert organization_id columns back to UUID
-- Note: This assumes all org_id values are valid UUIDs

-- System table
ALTER TABLE system
    ALTER COLUMN default_org_id TYPE UUID USING default_org_id::UUID;

-- Visitors table
ALTER TABLE visitors
    ALTER COLUMN organization_id TYPE UUID USING organization_id::UUID;

-- Payment providers table
ALTER TABLE payment_providers
    ALTER COLUMN organization_id TYPE UUID USING organization_id::UUID;

-- Projects table
ALTER TABLE projects
    ALTER COLUMN organization_id TYPE UUID USING organization_id::UUID;
