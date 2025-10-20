-- +goose Up
-- Create payment_providers table
CREATE TABLE payment_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Provider information
    provider_type VARCHAR(50) NOT NULL CHECK (provider_type IN ('stripe', 'paddle', 'paypal', 'lemon_squeezy', 'square')),

    -- Encrypted credentials
    api_key_encrypted TEXT NOT NULL,
    webhook_secret_encrypted TEXT NOT NULL,

    -- Status and metadata
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_synced_at TIMESTAMP NULL,

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_payment_providers_project_id ON payment_providers(project_id);
CREATE INDEX idx_payment_providers_organization_id ON payment_providers(organization_id);
CREATE INDEX idx_payment_providers_provider_type ON payment_providers(provider_type);
CREATE UNIQUE INDEX idx_payment_providers_project_provider ON payment_providers(project_id, provider_type) WHERE is_active = true;

-- Create trigger to update updated_at timestamp
CREATE TRIGGER update_payment_providers_updated_at BEFORE UPDATE ON payment_providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
-- Drop trigger and indexes
DROP TRIGGER IF EXISTS update_payment_providers_updated_at ON payment_providers;
DROP INDEX IF EXISTS idx_payment_providers_project_provider;
DROP INDEX IF EXISTS idx_payment_providers_provider_type;
DROP INDEX IF EXISTS idx_payment_providers_organization_id;
DROP INDEX IF EXISTS idx_payment_providers_project_id;

-- Drop table
DROP TABLE IF EXISTS payment_providers;
