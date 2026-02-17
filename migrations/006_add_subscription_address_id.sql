-- +goose Up
CREATE TABLE IF NOT EXISTS addresses (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    complex_id TEXT NOT NULL REFERENCES residential_complexes(id) ON DELETE CASCADE,
    city_id TEXT NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    address_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS address_id TEXT;

INSERT INTO addresses (id, user_id, name, complex_id, city_id, address_json, created_at)
SELECT
    s.id,
    s.user_id,
    COALESCE(s.address_name, s.address_json->>'name', 'Адрес'),
    s.complex_id,
    c.city_id,
    s.address_json,
    s.created_at
FROM subscriptions s
JOIN residential_complexes c ON c.id = s.complex_id
WHERE s.address_id IS NULL
ON CONFLICT (id) DO NOTHING;

UPDATE subscriptions
SET address_id = id
WHERE address_id IS NULL;

ALTER TABLE subscriptions
    ALTER COLUMN address_id SET NOT NULL;

ALTER TABLE subscriptions
    ADD CONSTRAINT IF NOT EXISTS subscriptions_address_id_fkey FOREIGN KEY (address_id) REFERENCES addresses(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_subscriptions_address_id ON subscriptions(address_id);

-- +goose Down
DROP INDEX IF EXISTS idx_subscriptions_address_id;

ALTER TABLE subscriptions
    DROP CONSTRAINT IF EXISTS subscriptions_address_id_fkey;

ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS address_id;

DROP TABLE IF EXISTS addresses;
