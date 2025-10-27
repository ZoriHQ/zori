-- +goose Up
-- Remove foreign key constraint from projects.organization_id
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_organization_id_fkey;

-- Remove foreign key constraint from payment_providers.organization_id
ALTER TABLE payment_providers DROP CONSTRAINT IF EXISTS payment_providers_organization_id_fkey;

-- Drop organization-related tables (we're using Stack Auth now)
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;
DROP INDEX IF EXISTS idx_organization_members_account_id;
DROP INDEX IF EXISTS idx_organization_members_org_id;
DROP TABLE IF EXISTS organization_members CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;

-- +goose Down
-- Recreate organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Recreate organization members table
CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, account_id)
);

-- Recreate indexes
CREATE INDEX idx_organization_members_org_id ON organization_members(organization_id);
CREATE INDEX idx_organization_members_account_id ON organization_members(account_id);

-- Recreate trigger
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Re-add foreign key constraint to projects.organization_id
ALTER TABLE projects
    ADD CONSTRAINT projects_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

-- Re-add foreign key constraint to payment_providers.organization_id
ALTER TABLE payment_providers
    ADD CONSTRAINT payment_providers_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;
