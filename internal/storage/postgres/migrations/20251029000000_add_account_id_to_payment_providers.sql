-- +goose Up
-- Add account_id column to payment_providers table
-- This represents the external account ID from the payment provider or auth system
ALTER TABLE payment_providers
ADD COLUMN account_id VARCHAR(255) NOT NULL DEFAULT '';

-- Create index for better performance when querying by account_id
CREATE INDEX idx_payment_providers_account_id ON payment_providers(account_id);

-- +goose Down
-- Drop index
DROP INDEX IF EXISTS idx_payment_providers_account_id;

-- Drop account_id column
ALTER TABLE payment_providers
DROP COLUMN IF EXISTS account_id;
