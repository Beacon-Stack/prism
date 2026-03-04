-- +goose Up
CREATE TABLE IF NOT EXISTS blocklist (
    id              TEXT PRIMARY KEY,
    movie_id        TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    release_guid    TEXT NOT NULL,
    release_title   TEXT NOT NULL,
    indexer_id      TEXT,
    protocol        TEXT NOT NULL DEFAULT '',
    size            INTEGER NOT NULL DEFAULT 0,
    added_at        DATETIME NOT NULL,
    notes           TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_blocklist_movie_id     ON blocklist(movie_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_blocklist_guid  ON blocklist(release_guid);

-- +goose Down
DROP TABLE IF EXISTS blocklist;
