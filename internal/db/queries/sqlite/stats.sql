-- name: InsertStorageSnapshot :exec
INSERT INTO storage_snapshots (id, captured_at, total_bytes, file_count)
VALUES (?, ?, ?, ?);

-- name: ListStorageSnapshots :many
SELECT * FROM storage_snapshots
ORDER BY captured_at DESC
LIMIT ?;

-- name: PruneOldStorageSnapshots :exec
DELETE FROM storage_snapshots
WHERE captured_at < ?;

-- name: GetCollectionStats :one
SELECT
    COUNT(*)                                                                                 AS total_movies,
    COALESCE(SUM(CASE WHEN monitored = TRUE THEN 1 ELSE 0 END), 0)                         AS monitored,
    COALESCE(SUM(CASE WHEN path IS NOT NULL AND path != '' THEN 1 ELSE 0 END), 0)          AS with_file,
    COALESCE(SUM(CASE WHEN monitored = TRUE AND (path IS NULL OR path = '') THEN 1 ELSE 0 END), 0) AS missing,
    COALESCE(SUM(CASE WHEN added_at > strftime('%Y-%m-%dT%H:%M:%SZ', 'now', 'start of day', '-30 days') THEN 1 ELSE 0 END), 0) AS recently_added
FROM movies;

-- name: GetStorageTotals :one
SELECT
    COALESCE(SUM(mf.size_bytes), 0) AS total_bytes,
    COUNT(*)                         AS file_count
FROM movie_files mf
JOIN movies m ON m.id = mf.movie_id;

-- name: ListMovieFileQualitiesWithIDs :many
SELECT mf.movie_id, mf.quality_json FROM movie_files mf
JOIN movies m ON m.id = mf.movie_id;

-- name: GetGrabStats :one
SELECT
    COUNT(*)                                                                           AS total_grabs,
    COALESCE(SUM(CASE WHEN download_status = 'completed' THEN 1 ELSE 0 END), 0)      AS successful,
    COALESCE(SUM(CASE WHEN download_status = 'failed' THEN 1 ELSE 0 END), 0)         AS failed
FROM grab_history;

-- name: GetTopIndexers :many
SELECT
    gh.indexer_id,
    COALESCE(ic.name, gh.indexer_id)                                                       AS indexer_name,
    COUNT(*)                                                                               AS grab_count,
    COALESCE(SUM(CASE WHEN gh.download_status = 'completed' THEN 1 ELSE 0 END), 0)       AS success_count
FROM grab_history gh
LEFT JOIN indexer_configs ic ON ic.id = gh.indexer_id
WHERE gh.indexer_id IS NOT NULL AND gh.indexer_id != ''
GROUP BY gh.indexer_id, ic.name
ORDER BY grab_count DESC
LIMIT 10;

-- name: GetMovieYearDistribution :many
SELECT year, COUNT(*) AS count
FROM movies
WHERE year > 0
GROUP BY year
ORDER BY year ASC;

-- name: GetMoviesAddedByMonth :many
SELECT
    strftime('%Y-%m', added_at) AS month,
    COUNT(*)                    AS count
FROM movies
WHERE added_at IS NOT NULL
GROUP BY strftime('%Y-%m', added_at)
ORDER BY month ASC;

-- name: ListMovieGenresJSON :many
SELECT genres_json FROM movies WHERE genres_json IS NOT NULL AND genres_json != '[]';

-- name: ListMovieFileStorage :many
-- One row per file with what the storage breakdown groups by.
SELECT m.id AS movie_id, m.title, m.year, m.runtime_minutes, m.genres_json, mf.size_bytes, mf.quality_json
FROM movie_files mf
JOIN movies m ON m.id = mf.movie_id;

-- name: GetWatchTotals :one
-- "Owned" is a movie with a file: the share watched is of what can be watched.
SELECT
    (SELECT COUNT(*) FROM movies WHERE path IS NOT NULL AND path != '') AS owned,
    (SELECT COUNT(*) FROM movies m
       WHERE m.path IS NOT NULL AND m.path != ''
         AND EXISTS (SELECT 1 FROM watch_history wh WHERE wh.movie_id = m.id)) AS watched,
    (SELECT COUNT(*) FROM watch_history) AS plays,
    (SELECT COALESCE(SUM(m.runtime_minutes), 0)
       FROM watch_history wh JOIN movies m ON m.id = wh.movie_id) AS minutes;

-- name: ListMostWatchedMovies :many
SELECT m.id, m.title, m.year, COUNT(*) AS plays, MAX(wh.watched_at) AS last_watched_at
FROM watch_history wh
JOIN movies m ON m.id = wh.movie_id
GROUP BY m.id, m.title, m.year
ORDER BY plays DESC, last_watched_at DESC
LIMIT ?;

-- name: GetWatchesByMonth :many
SELECT
    strftime('%Y-%m', watched_at) AS month,
    COUNT(*)                      AS plays
FROM watch_history
GROUP BY strftime('%Y-%m', watched_at)
ORDER BY month ASC;

-- name: ListLongestUnwatchedMovies :many
-- Movies with a file that nobody has watched, longest in the library first.
SELECT m.id, m.title, m.year, m.added_at
FROM movies m
WHERE m.path IS NOT NULL AND m.path != ''
  AND NOT EXISTS (SELECT 1 FROM watch_history wh WHERE wh.movie_id = m.id)
ORDER BY m.added_at ASC
LIMIT ?;
