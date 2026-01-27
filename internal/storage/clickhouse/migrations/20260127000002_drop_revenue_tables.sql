-- +goose Up
-- Drop revenue-related materialized views first (they depend on the tables)
DROP VIEW IF EXISTS revenue_timeline_daily_mv;
DROP VIEW IF EXISTS revenue_timeline_hourly_mv;
DROP VIEW IF EXISTS visitor_first_touch_attribution_mv;

-- Drop revenue-related tables
DROP TABLE IF EXISTS visitor_first_touch_attribution;
DROP TABLE IF EXISTS payment_events;

-- +goose Down
-- Data loss expected on rollback - tables would need to be recreated manually
