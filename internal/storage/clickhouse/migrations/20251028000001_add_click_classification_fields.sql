-- +goose Up
-- +goose StatementBegin
-- Add click element classification type
ALTER TABLE events ADD COLUMN IF NOT EXISTS click_element_type Nullable(String);
-- +goose StatementEnd

-- +goose StatementBegin
-- Add click element category
ALTER TABLE events ADD COLUMN IF NOT EXISTS click_element_category Nullable(String);
-- +goose StatementEnd

-- +goose StatementBegin
-- Add CTA click indicator
ALTER TABLE events ADD COLUMN IF NOT EXISTS is_cta_click Nullable(Bool);
-- +goose StatementEnd

-- +goose StatementBegin
-- Add link destination for link clicks
ALTER TABLE events ADD COLUMN IF NOT EXISTS link_destination Nullable(String);
-- +goose StatementEnd

-- +goose StatementBegin
-- Add external link indicator
ALTER TABLE events ADD COLUMN IF NOT EXISTS is_external_link Nullable(Bool);
-- +goose StatementEnd

-- +goose StatementBegin
-- Add download link indicator
ALTER TABLE events ADD COLUMN IF NOT EXISTS is_download_link Nullable(Bool);
-- +goose StatementEnd

-- +goose StatementBegin
-- Create index on click_element_type for fast filtering
CREATE INDEX IF NOT EXISTS idx_click_element_type ON events (click_element_type) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
-- Create index on click_element_category for analytics
CREATE INDEX IF NOT EXISTS idx_click_element_category ON events (click_element_category) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
-- Create index on is_cta_click for conversion tracking
CREATE INDEX IF NOT EXISTS idx_is_cta_click ON events (is_cta_click) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_is_cta_click ON events;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_click_element_category ON events;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_click_element_type ON events;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS is_download_link;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS is_external_link;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS link_destination;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS is_cta_click;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS click_element_category;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS click_element_type;
-- +goose StatementEnd
