-- +goose Up
ALTER TABLE grab_history ADD COLUMN score_breakdown TEXT NOT NULL DEFAULT '';

-- +goose Down
SELECT 1;
