-- +goose Up
CREATE TABLE IF NOT EXISTS time_windows (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE addresses
    ADD COLUMN IF NOT EXISTS time_window_id TEXT;

ALTER TABLE addresses
    ADD CONSTRAINT addresses_time_window_id_fkey FOREIGN KEY (time_window_id) REFERENCES time_windows(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_addresses_time_window_id ON addresses(time_window_id);
CREATE INDEX IF NOT EXISTS idx_time_windows_active ON time_windows(is_active);

-- +goose Down
DROP INDEX IF EXISTS idx_time_windows_active;
DROP INDEX IF EXISTS idx_addresses_time_window_id;

ALTER TABLE addresses
    DROP CONSTRAINT IF EXISTS addresses_time_window_id_fkey;

ALTER TABLE addresses
    DROP COLUMN IF EXISTS time_window_id;

DROP TABLE IF EXISTS time_windows;
