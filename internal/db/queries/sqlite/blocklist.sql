-- name: CreateBlocklistEntry :one
INSERT INTO blocklist (id, movie_id, release_guid, release_title, indexer_id,
    protocol, size, added_at, notes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: IsBlocklisted :one
SELECT COUNT(*) FROM blocklist WHERE release_guid = ?;

-- name: ListBlocklist :many
SELECT b.*, m.title AS movie_title
FROM blocklist b JOIN movies m ON m.id = b.movie_id
ORDER BY b.added_at DESC
LIMIT ? OFFSET ?;

-- name: CountBlocklist :one
SELECT COUNT(*) FROM blocklist;

-- name: DeleteBlocklistEntry :exec
DELETE FROM blocklist WHERE id = ?;

-- name: ClearBlocklist :exec
DELETE FROM blocklist;
