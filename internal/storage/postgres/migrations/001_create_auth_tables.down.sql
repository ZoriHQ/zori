-- Drop triggers
DROP TRIGGER IF EXISTS update_accounts_updated_at ON accounts;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_refresh_token;
DROP INDEX IF EXISTS idx_sessions_account_id;
DROP INDEX IF EXISTS idx_organization_members_account_id;
DROP INDEX IF EXISTS idx_organization_members_org_id;
DROP INDEX IF EXISTS idx_accounts_email;

-- Drop tables in reverse order (due to foreign key constraints)
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS organizations;
