-- +goose Up

-- Prevent duplicate active downloads for the same movie. Only one grab per
-- movie may be in a non-terminal state at any time. This is enforced at the
-- database level to avoid TOCTOU races between auto-search, manual grab, and
-- the RSS sync scheduler job.
CREATE UNIQUE INDEX IF NOT EXISTS idx_grab_history_active_movie
ON grab_history (movie_id)
WHERE download_status NOT IN ('completed', 'failed', 'removed');

-- +goose Down

DROP INDEX IF EXISTS idx_grab_history_active_movie;
