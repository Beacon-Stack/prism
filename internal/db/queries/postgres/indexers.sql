-- name: CreateIndexerConfig :one
INSERT INTO indexer_configs (id, name, kind, enabled, priority, settings, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetIndexerConfig :one
SELECT * FROM indexer_configs WHERE id = $1;

-- name: ListIndexerConfigs :many
SELECT * FROM indexer_configs ORDER BY priority ASC, name ASC;

-- name: ListEnabledIndexers :many
SELECT * FROM indexer_configs WHERE enabled = TRUE ORDER BY priority ASC, name ASC;

-- name: UpdateIndexerConfig :one
UPDATE indexer_configs SET
    name       = $1,
    kind       = $2,
    enabled    = $3,
    priority   = $4,
    settings   = $5,
    updated_at = $6
WHERE id = $7
RETURNING *;

-- name: DeleteIndexerConfig :exec
DELETE FROM indexer_configs WHERE id = $1;

-- name: CreateGrabHistory :one
INSERT INTO grab_history (
    id, movie_id, indexer_id, release_guid, release_title,
    release_source, release_resolution, release_codec, release_hdr,
    protocol, size, download_client_id, client_item_id, grabbed_at,
    download_status, downloaded_bytes, score_breakdown, release_edition
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13, $14,
    $15, $16, $17, $18
)
RETURNING *;

-- name: ListGrabHistoryByMovie :many
SELECT * FROM grab_history WHERE movie_id = $1 ORDER BY grabbed_at DESC;

-- name: ListGrabHistory :many
SELECT * FROM grab_history ORDER BY grabbed_at DESC LIMIT $1;

-- name: UpdateGrabDownloadClient :exec
UPDATE grab_history
SET download_client_id = $1, client_item_id = $2, download_status = 'queued'
WHERE id = $3;

-- name: UpdateGrabStatus :exec
UPDATE grab_history
SET download_status = $1, downloaded_bytes = $2
WHERE id = $3;

-- name: ListActiveGrabs :many
SELECT * FROM grab_history
WHERE client_item_id IS NOT NULL
  AND download_status NOT IN ('completed', 'failed', 'removed')
ORDER BY grabbed_at DESC;

-- name: GetGrabByClientItemID :one
SELECT * FROM grab_history
WHERE download_client_id = $1 AND client_item_id = $2
LIMIT 1;

-- name: MarkGrabRemoved :exec
UPDATE grab_history SET download_status = 'removed' WHERE id = $1;

-- name: ListGrabHistoryByStatus :many
SELECT * FROM grab_history WHERE download_status = $1 ORDER BY grabbed_at DESC LIMIT $2;

-- name: ListGrabHistoryByProtocol :many
SELECT * FROM grab_history WHERE protocol = $1 ORDER BY grabbed_at DESC LIMIT $2;

-- name: ListGrabHistoryByStatusAndProtocol :many
SELECT * FROM grab_history WHERE download_status = $1 AND protocol = $2 ORDER BY grabbed_at DESC LIMIT $3;

-- name: GetGrabByID :one
SELECT * FROM grab_history WHERE id = $1;
