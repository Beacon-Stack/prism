-- name: GetMediaManagement :one
SELECT * FROM media_management WHERE id = 1;

-- name: UpdateMediaManagement :one
UPDATE media_management
SET rename_movies            = ?,
    standard_movie_format    = ?,
    movie_folder_format      = ?,
    colon_replacement        = ?,
    import_extra_files       = ?,
    extra_file_extensions    = ?,
    unmonitor_deleted_movies = ?
WHERE id = 1
RETURNING *;
