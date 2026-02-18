-- +goose Up
-- This migration must be safe after 004/005 and for partially migrated databases.
-- Ensure address_id exists and is linked to addresses(id).
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS address_id TEXT;

-- Backfill address_id for legacy rows where address IDs matched subscription IDs.
UPDATE subscriptions s
SET address_id = a.id
FROM addresses a
WHERE s.address_id IS NULL
  AND a.id = s.id;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'subscriptions'
          AND column_name = 'address_id'
          AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE subscriptions
            ALTER COLUMN address_id SET NOT NULL;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'subscriptions_address_id_fkey'
          AND conrelid = 'subscriptions'::regclass
    ) THEN
        ALTER TABLE subscriptions
            ADD CONSTRAINT subscriptions_address_id_fkey
            FOREIGN KEY (address_id) REFERENCES addresses(id) ON DELETE CASCADE;
    END IF;
END $$;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_subscriptions_address_id ON subscriptions(address_id);

-- +goose Down
DROP INDEX IF EXISTS idx_subscriptions_address_id;
