-- +goose Up
ALTER TABLE movie_files ADD COLUMN mediainfo_json TEXT NOT NULL DEFAULT '';
ALTER TABLE movie_files ADD COLUMN mediainfo_scanned_at DATETIME;

-- +goose Down
SELECT 1;
