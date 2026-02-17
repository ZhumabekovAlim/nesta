-- +goose Up
CREATE TABLE IF NOT EXISTS addresses (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    complex_id TEXT NOT NULL REFERENCES residential_complexes(id) ON DELETE CASCADE,
    address_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_addresses_user ON addresses(user_id);

INSERT INTO addresses (id, user_id, name, complex_id, address_json, created_at)
SELECT
    id,
    user_id,
    COALESCE(address_name, address_json->>'name', 'Адрес'),
    complex_id,
    address_json,
    created_at
FROM subscriptions
ON CONFLICT (id) DO NOTHING;

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS address_id TEXT;

UPDATE subscriptions
SET address_id = id
WHERE address_id IS NULL;

ALTER TABLE subscriptions
    ALTER COLUMN address_id SET NOT NULL;

ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_address_id_fkey FOREIGN KEY (address_id) REFERENCES addresses(id) ON DELETE CASCADE;

DROP INDEX IF EXISTS idx_subscriptions_complex;

ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS address_name,
    DROP COLUMN IF EXISTS address_json,
    DROP COLUMN IF EXISTS complex_id;

-- +goose Down
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS complex_id TEXT,
    ADD COLUMN IF NOT EXISTS address_json JSONB,
    ADD COLUMN IF NOT EXISTS address_name TEXT;

UPDATE subscriptions s
SET complex_id = a.complex_id,
    address_json = a.address_json,
    address_name = a.name
FROM addresses a
WHERE s.address_id = a.id;

ALTER TABLE subscriptions
    ALTER COLUMN complex_id SET NOT NULL,
    ALTER COLUMN address_json SET NOT NULL,
    ALTER COLUMN address_name SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_subscriptions_complex ON subscriptions(complex_id);

ALTER TABLE subscriptions
    DROP CONSTRAINT IF EXISTS subscriptions_address_id_fkey;

ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS address_id;

DROP TABLE IF EXISTS addresses;
