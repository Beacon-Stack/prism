-- name: GetDownloadHandling :one
SELECT * FROM download_handling WHERE id = 1;

-- name: UpdateDownloadHandling :one
UPDATE download_handling
SET enable_completed              = ?,
    check_interval_minutes        = ?,
    redownload_failed             = ?,
    redownload_failed_interactive = ?
WHERE id = 1
RETURNING *;

-- name: ListRemotePathMappings :many
SELECT * FROM remote_path_mappings ORDER BY host, remote_path;

-- name: CreateRemotePathMapping :one
INSERT INTO remote_path_mappings (id, host, remote_path, local_path)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: DeleteRemotePathMapping :exec
DELETE FROM remote_path_mappings WHERE id = ?;
