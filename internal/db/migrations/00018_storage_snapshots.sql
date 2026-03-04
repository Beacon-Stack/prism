-- +goose Up
CREATE TABLE storage_snapshots (
    id           TEXT PRIMARY KEY,
    captured_at  DATETIME NOT NULL,
    total_bytes  INTEGER NOT NULL DEFAULT 0,
    file_count   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_storage_snapshots_captured_at ON storage_snapshots(captured_at);

-- +goose Down
DROP INDEX IF EXISTS idx_storage_snapshots_captured_at;
DROP TABLE IF EXISTS storage_snapshots;
