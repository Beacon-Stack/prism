-- +goose Up
ALTER TABLE collections ADD COLUMN total_items     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE collections ADD COLUMN in_library_items INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not support DROP COLUMN on older versions; recreate the table.
CREATE TABLE collections_old AS SELECT id, name, person_id, person_type, created_at FROM collections;
DROP TABLE collections;
ALTER TABLE collections_old RENAME TO collections;
CREATE UNIQUE INDEX idx_collections_person ON collections(person_id, person_type);
