-- +goose Up

-- Prevent duplicate active downloads for the same movie. Only one grab per
-- movie may be in a non-terminal state at any time. This is enforced at the
-- database level to avoid TOCTOU races between auto-search, manual grab, and
-- the RSS sync scheduler job.

-- First, clean up any existing duplicates: keep the most recent active grab
-- per movie and mark older ones as 'removed'.
UPDATE grab_history
SET download_status = 'removed'
WHERE id NOT IN (
    SELECT MAX(id) FROM grab_history
    WHERE download_status NOT IN ('completed', 'failed', 'removed')
    GROUP BY movie_id
)
AND download_status NOT IN ('completed', 'failed', 'removed');

CREATE UNIQUE INDEX IF NOT EXISTS idx_grab_history_active_movie
ON grab_history (movie_id)
WHERE download_status NOT IN ('completed', 'failed', 'removed');

-- +goose Down

DROP INDEX IF EXISTS idx_grab_history_active_movie;
