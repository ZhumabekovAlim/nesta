-- +goose Up
ALTER TABLE residential_complexes
    ALTER COLUMN threshold_n SET DEFAULT 30;

UPDATE residential_complexes
SET threshold_n = 30
WHERE threshold_n = 0;

-- +goose Down
UPDATE residential_complexes
SET threshold_n = 0
WHERE threshold_n = 30;

ALTER TABLE residential_complexes
    ALTER COLUMN threshold_n SET DEFAULT 0;
