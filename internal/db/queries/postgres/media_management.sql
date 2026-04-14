-- name: GetMediaManagement :one
SELECT * FROM media_management WHERE id = 1;

-- name: UpdateMediaManagement :one
UPDATE media_management
SET rename_movies            = $1,
    standard_movie_format    = $2,
    movie_folder_format      = $3,
    colon_replacement        = $4,
    import_extra_files       = $5,
    extra_file_extensions    = $6,
    unmonitor_deleted_movies = $7
WHERE id = 1
RETURNING *;
