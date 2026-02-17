-- +goose Up
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS address_name TEXT;

UPDATE subscriptions
SET address_name = COALESCE(address_json->>'name', 'Адрес')
WHERE address_name IS NULL;

ALTER TABLE subscriptions
    ALTER COLUMN address_name SET NOT NULL;

-- +goose Down
ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS address_name;
