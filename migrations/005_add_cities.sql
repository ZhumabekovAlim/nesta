-- +goose Up
CREATE TABLE IF NOT EXISTS cities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO cities (id, name)
SELECT DISTINCT
    regexp_replace(lower(city), '\\s+', '_', 'g'),
    city
FROM residential_complexes
WHERE city IS NOT NULL AND city != ''
ON CONFLICT (name) DO NOTHING;

ALTER TABLE residential_complexes
    ADD COLUMN IF NOT EXISTS city_id TEXT;

UPDATE residential_complexes
SET city_id = COALESCE(
    city_id,
    regexp_replace(lower(city), '\\s+', '_', 'g')
)
WHERE city_id IS NULL;

ALTER TABLE residential_complexes
    ALTER COLUMN city_id SET NOT NULL;

ALTER TABLE residential_complexes
    ADD CONSTRAINT residential_complexes_city_id_fkey FOREIGN KEY (city_id) REFERENCES cities(id) ON DELETE RESTRICT;

ALTER TABLE addresses
    ADD COLUMN IF NOT EXISTS city_id TEXT;

UPDATE addresses a
SET city_id = complexes.city_id
FROM residential_complexes complexes
WHERE a.complex_id = complexes.id
  AND a.city_id IS NULL;

ALTER TABLE addresses
    ALTER COLUMN city_id SET NOT NULL;

ALTER TABLE addresses
    ADD CONSTRAINT addresses_city_id_fkey FOREIGN KEY (city_id) REFERENCES cities(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_addresses_city ON addresses(city_id);
CREATE INDEX IF NOT EXISTS idx_complexes_city_id ON residential_complexes(city_id);

-- +goose Down
DROP INDEX IF EXISTS idx_complexes_city_id;
DROP INDEX IF EXISTS idx_addresses_city;

ALTER TABLE addresses
    DROP CONSTRAINT IF EXISTS addresses_city_id_fkey;

ALTER TABLE addresses
    DROP COLUMN IF EXISTS city_id;

ALTER TABLE residential_complexes
    DROP CONSTRAINT IF EXISTS residential_complexes_city_id_fkey;

ALTER TABLE residential_complexes
    DROP COLUMN IF EXISTS city_id;

DROP TABLE IF EXISTS cities;
