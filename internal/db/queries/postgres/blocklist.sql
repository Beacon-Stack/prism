-- name: CreateBlocklistEntry :one
INSERT INTO blocklist (id, movie_id, release_guid, release_title, indexer_id,
    protocol, size, added_at, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: IsBlocklisted :one
SELECT COUNT(*) FROM blocklist WHERE release_guid = $1;

-- name: ListBlocklist :many
SELECT b.*, m.title AS movie_title
FROM blocklist b JOIN movies m ON m.id = b.movie_id
ORDER BY b.added_at DESC
LIMIT $1 OFFSET $2;

-- name: CountBlocklist :one
SELECT COUNT(*) FROM blocklist;

-- name: DeleteBlocklistEntry :exec
DELETE FROM blocklist WHERE id = $1;

-- name: ClearBlocklist :exec
DELETE FROM blocklist;

-- name: IsBlocklistedByTitle :one
SELECT COUNT(*) FROM blocklist WHERE release_title = $1;
