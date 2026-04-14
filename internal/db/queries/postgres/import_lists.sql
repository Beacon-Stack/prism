-- Import list configs

-- name: CreateImportListConfig :one
INSERT INTO import_list_configs (
    id, name, kind, enabled, settings, search_on_add, monitor,
    min_availability, quality_profile_id, library_id, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetImportListConfig :one
SELECT * FROM import_list_configs WHERE id = $1;

-- name: ListImportListConfigs :many
SELECT * FROM import_list_configs ORDER BY name ASC;

-- name: ListEnabledImportLists :many
SELECT * FROM import_list_configs WHERE enabled = TRUE ORDER BY name ASC;

-- name: UpdateImportListConfig :one
UPDATE import_list_configs SET
    name               = $1,
    kind               = $2,
    enabled            = $3,
    settings           = $4,
    search_on_add      = $5,
    monitor            = $6,
    min_availability   = $7,
    quality_profile_id = $8,
    library_id         = $9,
    updated_at         = $10
WHERE id = $11
RETURNING *;

-- name: DeleteImportListConfig :exec
DELETE FROM import_list_configs WHERE id = $1;

-- Import exclusions

-- name: CreateImportExclusion :one
INSERT INTO import_exclusions (id, tmdb_id, title, year, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetImportExclusionByTMDBID :one
SELECT * FROM import_exclusions WHERE tmdb_id = $1;

-- name: ListImportExclusions :many
SELECT * FROM import_exclusions ORDER BY title ASC;

-- name: ListExcludedTMDBIDs :many
SELECT tmdb_id FROM import_exclusions;

-- name: DeleteImportExclusion :exec
DELETE FROM import_exclusions WHERE id = $1;
