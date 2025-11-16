-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS visitor_attribution
(
    visitor_id String,
    first_seen_at DateTime64(3, 'UTC'),
    last_seen_at DateTime64(3, 'UTC'),
    first_referrer_domain Nullable(String),
    last_referrer_domain Nullable(String),
    project_id String,
    organization_id String,
    updated_at DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (organization_id, project_id, visitor_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Step 2: Create materialized view to update attribution
CREATE MATERIALIZED VIEW IF NOT EXISTS visitor_attribution_mv
TO visitor_attribution
AS
SELECT
    visitor_id,
    min(client_timestamp_utc) AS first_seen_at,
    max(client_timestamp_utc) AS last_seen_at,
    argMin(referrer_domain, client_timestamp_utc) AS first_referrer_domain,
    argMax(referrer_domain, client_timestamp_utc) AS last_referrer_domain,
    project_id,
    organization_id,
    now() AS updated_at
FROM events
WHERE visitor_id != ''
GROUP BY visitor_id, project_id, organization_id;
-- +goose StatementEnd

-- +goose StatementBegin
-- Step 3: Create revenue attribution view
CREATE VIEW IF NOT EXISTS revenue_by_source AS
SELECT
    ifNull(va.last_referrer_domain, 'DIRECT/NONE') AS traffic_origin,
    count(DISTINCT pe.payment_id) AS total_payments,
    count(DISTINCT pe.visitor_id) AS paying_visitors,
    sum(pe.amount) AS total_revenue,
    avg(pe.amount) AS avg_payment_amount,
    round(sum(pe.amount) / count(DISTINCT pe.visitor_id), 2) AS revenue_per_visitor,
    pe.project_id,
    pe.organization_id
FROM payment_events pe
LEFT JOIN visitor_attribution va
    ON pe.visitor_id = va.visitor_id
    AND pe.project_id = va.project_id
    AND pe.organization_id = va.organization_id
WHERE  pe.visitor_id IS NOT NULL
GROUP BY traffic_origin, pe.project_id, pe.organization_id;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS revenue_by_source;
-- +goose StatementEnd

-- +goose StatementBegin
DROP MATERIALIZED VIEW IF EXISTS visitor_attribution_mv;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS visitor_attribution;
-- +goose StatementEnd
