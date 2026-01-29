-- +goose Up
ALTER TABLE residential_complexes
    ADD COLUMN IF NOT EXISTS address TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_residential_complexes_address
    ON residential_complexes(address)
    WHERE address IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS uniq_residential_complexes_address;
ALTER TABLE residential_complexes
    DROP COLUMN IF EXISTS address;
