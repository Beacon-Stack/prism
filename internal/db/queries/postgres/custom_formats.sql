-- name: ListCustomFormats :many
SELECT * FROM custom_formats ORDER BY name ASC;

-- name: GetCustomFormat :one
SELECT * FROM custom_formats WHERE id = $1;

-- name: CreateCustomFormat :one
INSERT INTO custom_formats (id, name, include_when_renaming, specifications_json, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateCustomFormat :one
UPDATE custom_formats
SET name = $1, include_when_renaming = $2, specifications_json = $3, updated_at = $4
WHERE id = $5
RETURNING *;

-- name: DeleteCustomFormat :exec
DELETE FROM custom_formats WHERE id = $1;

-- name: ListCustomFormatScores :many
SELECT * FROM custom_format_scores WHERE quality_profile_id = $1;

-- name: SetCustomFormatScore :exec
INSERT INTO custom_format_scores (quality_profile_id, custom_format_id, score)
VALUES ($1, $2, $3)
ON CONFLICT (quality_profile_id, custom_format_id) DO UPDATE SET score = excluded.score;

-- name: DeleteCustomFormatScores :exec
DELETE FROM custom_format_scores WHERE quality_profile_id = $1;
