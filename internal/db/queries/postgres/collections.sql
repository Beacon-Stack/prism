-- name: CreateCollection :one
INSERT INTO collections (id, name, person_id, person_type, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListCollections :many
SELECT * FROM collections ORDER BY name ASC;

-- name: GetCollection :one
SELECT * FROM collections WHERE id = $1;

-- name: GetCollectionByPerson :one
SELECT * FROM collections WHERE person_id = $1 AND person_type = $2;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = $1;

-- name: UpdateCollectionCounts :exec
UPDATE collections SET total_items = $1, in_library_items = $2 WHERE id = $3;
