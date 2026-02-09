-- +goose Up
CREATE TABLE IF NOT EXISTS subscription_types (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    subtitle TEXT,
    price_cents INT NOT NULL DEFAULT 0,
    features JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscription_types_active ON subscription_types(is_active);

-- +goose Down
DROP TABLE IF EXISTS subscription_types;
