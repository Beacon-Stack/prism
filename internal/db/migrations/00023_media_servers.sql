-- +goose Up

CREATE TABLE IF NOT EXISTS media_server_configs (
    id         TEXT PRIMARY KEY,
    name       TEXT    NOT NULL,
    kind       TEXT    NOT NULL,             -- "plex", "emby", "jellyfin"
    enabled    INTEGER NOT NULL DEFAULT 1,
    settings   TEXT    NOT NULL DEFAULT '{}', -- JSON: plugin-specific settings
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL
);

-- +goose Down

DROP TABLE IF EXISTS media_server_configs;
