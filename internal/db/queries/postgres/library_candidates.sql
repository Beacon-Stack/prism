-- name: UpsertLibraryFileCandidate :exec
INSERT INTO library_file_candidates
    (library_id, file_path, file_size, parsed_title, parsed_year, scanned_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT(library_id, file_path) DO UPDATE SET
    file_size    = excluded.file_size,
    parsed_title = excluded.parsed_title,
    parsed_year  = excluded.parsed_year,
    scanned_at   = excluded.scanned_at;

-- name: SetLibraryFileCandidateMatch :exec
UPDATE library_file_candidates
SET tmdb_id             = $1,
    tmdb_title          = $2,
    tmdb_year           = $3,
    tmdb_original_title = $4,
    auto_matched        = TRUE,
    matched_at          = $5
WHERE library_id = $6 AND file_path = $7;

-- name: ListLibraryFileCandidates :many
SELECT * FROM library_file_candidates WHERE library_id = $1;

-- name: ListUnmatchedLibraryFileCandidates :many
SELECT * FROM library_file_candidates
WHERE library_id = $1 AND tmdb_id = 0 AND parsed_year > 0 AND parsed_title != '';

-- name: DeleteLibraryFileCandidate :exec
DELETE FROM library_file_candidates WHERE library_id = $1 AND file_path = $2;

-- name: PruneStaleLibraryFileCandidates :exec
-- Removes candidates that were not seen in the current scan (scanned_at < cutoff).
DELETE FROM library_file_candidates WHERE library_id = $1 AND scanned_at < $2;
