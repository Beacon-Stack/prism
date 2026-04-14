-- name: CreateMovie :one
INSERT INTO movies (
    id, tmdb_id, imdb_id, title, original_title,
    year, overview, runtime_minutes, genres_json,
    poster_url, fanart_url, status, monitored,
    library_id, quality_profile_id, path,
    added_at, updated_at, metadata_refreshed_at,
    minimum_availability, release_date
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15, $16,
    $17, $18, $19,
    $20, $21
)
RETURNING *;

-- name: GetMovie :one
SELECT * FROM movies WHERE id = $1;

-- name: GetMovieByTMDBID :one
SELECT * FROM movies WHERE tmdb_id = $1;

-- name: ListMovies :many
SELECT * FROM movies
ORDER BY title ASC
LIMIT $1 OFFSET $2;

-- name: ListMoviesByLibrary :many
SELECT * FROM movies
WHERE library_id = $1
ORDER BY title ASC
LIMIT $2 OFFSET $3;

-- name: ListMonitoredMovies :many
SELECT * FROM movies
WHERE monitored = TRUE
ORDER BY title ASC;

-- name: CountMovies :one
SELECT COUNT(*) FROM movies;

-- name: CountMoviesByLibrary :one
SELECT COUNT(*) FROM movies WHERE library_id = $1;

-- name: UpdateMovie :one
UPDATE movies SET
    title                = $1,
    original_title       = $2,
    year                 = $3,
    overview             = $4,
    runtime_minutes      = $5,
    genres_json          = $6,
    poster_url           = $7,
    fanart_url           = $8,
    status               = $9,
    monitored            = $10,
    library_id           = $11,
    quality_profile_id   = $12,
    minimum_availability = $13,
    release_date         = $14,
    updated_at           = $15
WHERE id = $16
RETURNING *;

-- name: UpdateMovieTMDBID :exec
UPDATE movies SET tmdb_id = $1, updated_at = $2 WHERE id = $3;

-- name: UpdateMovieStatus :one
UPDATE movies SET status = $1, updated_at = $2 WHERE id = $3 RETURNING *;

-- name: UpdateMoviePath :one
UPDATE movies SET path = $1, updated_at = $2 WHERE id = $3 RETURNING *;

-- name: UpdateMovieMetadataRefreshed :exec
UPDATE movies SET metadata_refreshed_at = $1, updated_at = $2 WHERE id = $3;

-- name: DeleteMovie :exec
DELETE FROM movies WHERE id = $1;

-- name: CreateMovieFile :one
INSERT INTO movie_files (
    id, movie_id, path, size_bytes, quality_json,
    edition, imported_at, indexed_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8
)
RETURNING *;

-- name: GetMovieFile :one
SELECT * FROM movie_files WHERE id = $1;

-- name: ListMovieFiles :many
SELECT * FROM movie_files WHERE movie_id = $1 ORDER BY imported_at DESC;

-- name: UpdateMovieFileIndexed :exec
UPDATE movie_files SET indexed_at = $1 WHERE id = $2;

-- name: UpdateMovieFilePath :exec
UPDATE movie_files SET path = $1 WHERE id = $2;

-- name: DeleteMovieFile :exec
DELETE FROM movie_files WHERE id = $1;

-- name: SumMovieFileSizesByLibrary :one
SELECT COALESCE(SUM(mf.size_bytes), 0)
FROM movie_files mf
JOIN movies m ON m.id = mf.movie_id
WHERE m.library_id = $1;

-- name: ListMovieFilesByLibrary :many
SELECT mf.*
FROM movie_files mf
JOIN movies m ON m.id = mf.movie_id
WHERE m.library_id = $1
ORDER BY mf.path ASC;

-- name: GetMovieFileByPath :one
SELECT * FROM movie_files WHERE path = $1;

-- name: ListMonitoredMoviesWithoutFile :many
SELECT m.*
FROM movies m
LEFT JOIN movie_files mf ON mf.movie_id = m.id
WHERE m.monitored = TRUE
  AND mf.id IS NULL
ORDER BY m.title ASC
LIMIT $1 OFFSET $2;

-- name: CountMonitoredMoviesWithoutFile :one
SELECT COUNT(*)
FROM movies m
LEFT JOIN movie_files mf ON mf.movie_id = m.id
WHERE m.monitored = TRUE
  AND mf.id IS NULL;

-- name: ListMonitoredMoviesWithFiles :many
SELECT m.*, mf.quality_json, qp.cutoff_json
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
JOIN quality_profiles qp ON qp.id = m.quality_profile_id
WHERE m.monitored = TRUE
ORDER BY m.title ASC;

-- name: UpdateMovieFileMediainfo :exec
UPDATE movie_files
SET mediainfo_json       = $1,
    mediainfo_scanned_at = $2
WHERE id = $3;

-- name: ListUnscannedMovieFiles :many
SELECT id, path FROM movie_files
WHERE mediainfo_json = ''
ORDER BY imported_at DESC;

-- name: UpdateMoviePreferredEdition :exec
UPDATE movies SET preferred_edition = $1, updated_at = $2 WHERE id = $3;

-- name: ListMoviesWithEditionMismatch :many
SELECT m.id, m.title, m.year, m.preferred_edition, mf.edition as file_edition
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
WHERE m.preferred_edition IS NOT NULL
  AND m.preferred_edition != ''
  AND (mf.edition IS NULL OR mf.edition != m.preferred_edition)
ORDER BY m.title ASC
LIMIT $1 OFFSET $2;

-- name: CountEditionMismatches :one
SELECT COUNT(*)
FROM movies m
JOIN movie_files mf ON mf.movie_id = m.id
WHERE m.preferred_edition IS NOT NULL
  AND m.preferred_edition != ''
  AND (mf.edition IS NULL OR mf.edition != m.preferred_edition);

-- name: UpdateMovieFileEdition :exec
UPDATE movie_files SET edition = $1 WHERE id = $2;

-- name: ListAllTMDBIDs :many
SELECT tmdb_id FROM movies WHERE tmdb_id != 0;

-- name: ListMovieSummaries :many
SELECT id, tmdb_id, title, year, status FROM movies WHERE tmdb_id != 0;
