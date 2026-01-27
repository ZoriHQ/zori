-- +goose Up
DROP TABLE IF EXISTS visitors CASCADE;
DROP TABLE IF EXISTS payment_providers CASCADE;

-- +goose Down
-- Data loss expected on rollback - tables would need to be recreated manually
