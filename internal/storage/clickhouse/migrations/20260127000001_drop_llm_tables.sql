-- +goose Up
-- Drop LLM materialized views first (they depend on the tables)
DROP VIEW IF EXISTS llm_daily_costs_mv;
DROP VIEW IF EXISTS llm_hourly_throughput_mv;

-- Drop LLM tables
DROP TABLE IF EXISTS llm_generations;
DROP TABLE IF EXISTS llm_traces;

-- +goose Down
-- Data loss expected on rollback - tables would need to be recreated manually
-- See 20251127000001_create_llm_tables.sql and 20251127000002_create_llm_materialized_views.sql for original schema
