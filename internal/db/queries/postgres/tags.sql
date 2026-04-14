-- name: CreateTag :one
INSERT INTO tags (id, name) VALUES ($1, $2) RETURNING *;

-- name: GetTag :one
SELECT * FROM tags WHERE id = $1;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = $1;

-- name: ListTags :many
SELECT * FROM tags ORDER BY name ASC;

-- name: UpdateTag :one
UPDATE tags SET name = $1 WHERE id = $2 RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;

-- Tag counts per entity type (for usage display).

-- name: CountMoviesForTag :one
SELECT COUNT(*) FROM movie_tags WHERE tag_id = $1;

-- name: CountIndexersForTag :one
SELECT COUNT(*) FROM indexer_tags WHERE tag_id = $1;

-- name: CountDownloadClientsForTag :one
SELECT COUNT(*) FROM download_client_tags WHERE tag_id = $1;

-- name: CountNotificationsForTag :one
SELECT COUNT(*) FROM notification_tags WHERE tag_id = $1;

-- Movie tag operations.

-- name: SetMovieTags :exec
DELETE FROM movie_tags WHERE movie_id = $1;

-- name: AddMovieTag :exec
INSERT INTO movie_tags (movie_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListMovieTagIDs :many
SELECT tag_id FROM movie_tags WHERE movie_id = $1;

-- Indexer tag operations.

-- name: SetIndexerTags :exec
DELETE FROM indexer_tags WHERE indexer_id = $1;

-- name: AddIndexerTag :exec
INSERT INTO indexer_tags (indexer_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListIndexerTagIDs :many
SELECT tag_id FROM indexer_tags WHERE indexer_id = $1;

-- Download client tag operations.

-- name: SetDownloadClientTags :exec
DELETE FROM download_client_tags WHERE download_client_id = $1;

-- name: AddDownloadClientTag :exec
INSERT INTO download_client_tags (download_client_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListDownloadClientTagIDs :many
SELECT tag_id FROM download_client_tags WHERE download_client_id = $1;

-- Notification tag operations.

-- name: SetNotificationTags :exec
DELETE FROM notification_tags WHERE notification_id = $1;

-- name: AddNotificationTag :exec
INSERT INTO notification_tags (notification_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListNotificationTagIDs :many
SELECT tag_id FROM notification_tags WHERE notification_id = $1;

-- Import list tag operations.

-- name: CountImportListsForTag :one
SELECT COUNT(*) FROM import_list_tags WHERE tag_id = $1;

-- name: SetImportListTags :exec
DELETE FROM import_list_tags WHERE import_list_id = $1;

-- name: AddImportListTag :exec
INSERT INTO import_list_tags (import_list_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListImportListTagIDs :many
SELECT tag_id FROM import_list_tags WHERE import_list_id = $1;
