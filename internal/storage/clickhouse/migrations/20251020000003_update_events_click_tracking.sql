-- +goose Up
-- +goose StatementBegin
-- Add new click_element fields
ALTER TABLE events ADD COLUMN IF NOT EXISTS click_element_tag Nullable(String);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS click_element_selector Nullable(String);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS click_element_text Nullable(String);
-- +goose StatementEnd

-- +goose StatementBegin
-- Add screen dimensions for click position context
ALTER TABLE events ADD COLUMN IF NOT EXISTS click_screen_width Nullable(UInt16);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events ADD COLUMN IF NOT EXISTS click_screen_height Nullable(UInt16);
-- +goose StatementEnd

-- +goose StatementBegin
-- Add host column if it doesn't exist (it should exist from previous migration)
ALTER TABLE events ADD COLUMN IF NOT EXISTS host String DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
-- Create index on click_element_selector for funnel and conversion tracking
CREATE INDEX IF NOT EXISTS idx_click_element_selector ON events (click_element_selector) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose StatementBegin
-- Create index on host for multi-domain tracking
CREATE INDEX IF NOT EXISTS idx_host ON events (host) TYPE bloom_filter GRANULARITY 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_host ON events;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_click_element_selector ON events;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS click_screen_height;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS click_screen_width;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS click_element_text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS click_element_selector;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE events DROP COLUMN IF EXISTS click_element_tag;
-- +goose StatementEnd
