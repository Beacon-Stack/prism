-- name: InsertWatchEvent :exec
INSERT INTO watch_history (id, movie_id, tmdb_id, watched_at, user_name, source)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT DO NOTHING;

-- name: WatchStatusForMovie :one
SELECT
    COUNT(*) AS play_count,
    MAX(watched_at) AS last_watched_at,
    MIN(watched_at) AS first_watched_at
FROM watch_history
WHERE movie_id = ?;

-- name: WatchStats :one
SELECT
    (SELECT COUNT(DISTINCT movie_id) FROM watch_history) AS watched_count,
    (SELECT COUNT(*) FROM movies) AS total_count;

-- name: GetSyncState :one
SELECT last_sync_at FROM watch_sync_state WHERE media_server_id = ?;

-- name: UpsertSyncState :exec
INSERT INTO watch_sync_state (media_server_id, last_sync_at)
VALUES (?, ?)
ON CONFLICT(media_server_id) DO UPDATE SET last_sync_at = excluded.last_sync_at;

