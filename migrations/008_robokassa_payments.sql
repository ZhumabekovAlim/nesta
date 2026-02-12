-- +goose Up
CREATE SEQUENCE IF NOT EXISTS robokassa_inv_id_seq START WITH 100000;

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS inv_id BIGINT,
    ADD COLUMN IF NOT EXISTS user_id TEXT,
    ADD COLUMN IF NOT EXISTS order_id TEXT,
    ADD COLUMN IF NOT EXISTS subscription_id TEXT,
    ADD COLUMN IF NOT EXISTS amount_value NUMERIC(18,6),
    ADD COLUMN IF NOT EXISTS currency TEXT,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS raw_init_payload JSONB,
    ADD COLUMN IF NOT EXISTS raw_callback_payload JSONB;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_payments_robokassa_inv_id ON payments(provider, inv_id) WHERE inv_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS payment_events (
    id BIGSERIAL PRIMARY KEY,
    payment_id TEXT NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_events_payment_id ON payment_events(payment_id);

-- +goose Down
DROP INDEX IF EXISTS idx_payment_events_payment_id;
DROP TABLE IF EXISTS payment_events;
DROP INDEX IF EXISTS uniq_payments_robokassa_inv_id;
ALTER TABLE payments
    DROP COLUMN IF EXISTS raw_callback_payload,
    DROP COLUMN IF EXISTS raw_init_payload,
    DROP COLUMN IF EXISTS paid_at,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS amount_value,
    DROP COLUMN IF EXISTS subscription_id,
    DROP COLUMN IF EXISTS order_id,
    DROP COLUMN IF EXISTS user_id,
    DROP COLUMN IF EXISTS inv_id;
DROP SEQUENCE IF EXISTS robokassa_inv_id_seq;
