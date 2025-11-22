-- +goose Up
CREATE TABLE funnels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    organization_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    funnel_type VARCHAR(20) NOT NULL DEFAULT 'sequential',
    window_seconds INTEGER NOT NULL DEFAULT 86400,
    is_strict BOOLEAN NOT NULL DEFAULT false,
    depth_config JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, name)
);

CREATE INDEX idx_funnels_project_id ON funnels(project_id);
CREATE INDEX idx_funnels_organization_id ON funnels(organization_id);

CREATE TRIGGER update_funnels_updated_at BEFORE UPDATE ON funnels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE funnel_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    funnel_id UUID NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    step_order INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    condition JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(funnel_id, step_order)
);

CREATE INDEX idx_funnel_steps_funnel_id ON funnel_steps(funnel_id);

-- +goose Down
DROP TRIGGER IF EXISTS update_funnels_updated_at ON funnels;
DROP INDEX IF EXISTS idx_funnel_steps_funnel_id;
DROP INDEX IF EXISTS idx_funnels_organization_id;
DROP INDEX IF EXISTS idx_funnels_project_id;
DROP TABLE IF EXISTS funnel_steps;
DROP TABLE IF EXISTS funnels;
