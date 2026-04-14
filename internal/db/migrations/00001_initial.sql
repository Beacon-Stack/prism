-- +goose Up

-- ── Quality profiles ────────────────────────────────────────────────────────

CREATE TABLE quality_profiles (
    id                      TEXT NOT NULL PRIMARY KEY,
    name                    TEXT NOT NULL,
    cutoff_json             TEXT NOT NULL DEFAULT '{}',
    qualities_json          TEXT NOT NULL DEFAULT '[]',
    upgrade_allowed         BOOLEAN NOT NULL DEFAULT TRUE,
    upgrade_until_json      TEXT,
    created_at              TEXT NOT NULL,
    updated_at              TEXT NOT NULL,
    min_custom_format_score INTEGER NOT NULL DEFAULT 0,
    upgrade_until_cf_score  INTEGER NOT NULL DEFAULT 0,
    row_id                  SERIAL
);

CREATE UNIQUE INDEX quality_profiles_name_unique ON quality_profiles(name);

-- ── Libraries ───────────────────────────────────────────────────────────────

CREATE TABLE libraries (
    id                          TEXT NOT NULL PRIMARY KEY,
    name                        TEXT NOT NULL,
    root_path                   TEXT NOT NULL,
    default_quality_profile_id  TEXT NOT NULL REFERENCES quality_profiles(id),
    naming_format               TEXT,
    min_free_space_gb           INTEGER NOT NULL DEFAULT 5,
    tags_json                   TEXT NOT NULL DEFAULT '[]',
    created_at                  TEXT NOT NULL,
    updated_at                  TEXT NOT NULL,
    folder_format               TEXT,
    row_id                      SERIAL
);

-- ── Movies ──────────────────────────────────────────────────────────────────

CREATE TABLE movies (
    id                      TEXT NOT NULL PRIMARY KEY,
    tmdb_id                 INTEGER NOT NULL DEFAULT 0,
    imdb_id                 TEXT,
    title                   TEXT NOT NULL,
    original_title          TEXT NOT NULL,
    year                    INTEGER NOT NULL,
    overview                TEXT NOT NULL DEFAULT '',
    runtime_minutes         INTEGER,
    genres_json             TEXT NOT NULL DEFAULT '[]',
    poster_url              TEXT,
    fanart_url              TEXT,
    status                  TEXT NOT NULL DEFAULT 'announced',
    monitored               BOOLEAN NOT NULL DEFAULT TRUE,
    library_id              TEXT NOT NULL REFERENCES libraries(id),
    quality_profile_id      TEXT NOT NULL REFERENCES quality_profiles(id),
    path                    TEXT,
    added_at                TEXT NOT NULL,
    updated_at              TEXT NOT NULL,
    metadata_refreshed_at   TEXT,
    minimum_availability    TEXT NOT NULL DEFAULT 'released',
    release_date            TEXT NOT NULL DEFAULT '',
    preferred_edition       TEXT,
    row_id                  SERIAL
);

CREATE UNIQUE INDEX movies_tmdb_id_unique ON movies(tmdb_id) WHERE tmdb_id != 0;
CREATE INDEX movies_library_id            ON movies(library_id);
CREATE INDEX movies_status                ON movies(status);
CREATE INDEX movies_monitored             ON movies(monitored);

-- ── Movie files ─────────────────────────────────────────────────────────────

CREATE TABLE movie_files (
    id                   TEXT NOT NULL PRIMARY KEY,
    movie_id             TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    path                 TEXT NOT NULL UNIQUE,
    size_bytes           BIGINT NOT NULL,
    quality_json         TEXT NOT NULL,
    edition              TEXT,
    imported_at          TEXT NOT NULL,
    indexed_at           TEXT NOT NULL,
    mediainfo_json       TEXT NOT NULL DEFAULT '',
    mediainfo_scanned_at TIMESTAMPTZ
);

CREATE INDEX movie_files_movie_id ON movie_files(movie_id);
CREATE INDEX movie_files_edition  ON movie_files(edition);

-- ── Indexer configs ─────────────────────────────────────────────────────────

CREATE TABLE indexer_configs (
    id          TEXT PRIMARY KEY,
    name        TEXT    NOT NULL,
    kind        TEXT    NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    priority    INTEGER NOT NULL DEFAULT 25,
    settings    TEXT    NOT NULL DEFAULT '{}',
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);

-- ── Grab history ────────────────────────────────────────────────────────────

CREATE TABLE grab_history (
    id                  TEXT    PRIMARY KEY,
    movie_id            TEXT    NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    indexer_id          TEXT,
    release_guid        TEXT    NOT NULL,
    release_title       TEXT    NOT NULL,
    release_source      TEXT    NOT NULL DEFAULT 'unknown',
    release_resolution  TEXT    NOT NULL DEFAULT 'unknown',
    release_codec       TEXT    NOT NULL DEFAULT 'unknown',
    release_hdr         TEXT    NOT NULL DEFAULT 'none',
    protocol            TEXT    NOT NULL DEFAULT 'unknown',
    size                BIGINT  NOT NULL DEFAULT 0,
    download_client_id  TEXT,
    client_item_id      TEXT,
    grabbed_at          TEXT    NOT NULL,
    download_status     TEXT    NOT NULL DEFAULT 'queued',
    downloaded_bytes    BIGINT  NOT NULL DEFAULT 0,
    score_breakdown     TEXT    NOT NULL DEFAULT '',
    release_edition     TEXT
);

CREATE INDEX idx_grab_history_movie_id   ON grab_history(movie_id);
CREATE INDEX idx_grab_history_grabbed_at ON grab_history(grabbed_at DESC);
CREATE INDEX idx_grab_history_status     ON grab_history(download_status);

CREATE UNIQUE INDEX idx_grab_history_active_movie
    ON grab_history (movie_id)
    WHERE download_status NOT IN ('completed', 'failed', 'removed');

-- ── Download client configs ─────────────────────────────────────────────────

CREATE TABLE download_client_configs (
    id          TEXT PRIMARY KEY,
    name        TEXT    NOT NULL,
    kind        TEXT    NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    priority    INTEGER NOT NULL DEFAULT 25,
    settings    TEXT    NOT NULL DEFAULT '{}',
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);

-- ── Notification configs ────────────────────────────────────────────────────

CREATE TABLE notification_configs (
    id         TEXT PRIMARY KEY,
    name       TEXT    NOT NULL,
    kind       TEXT    NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    settings   TEXT    NOT NULL DEFAULT '{}',
    on_events  TEXT    NOT NULL DEFAULT '[]',
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL
);

-- ── Blocklist ───────────────────────────────────────────────────────────────

CREATE TABLE blocklist (
    id              TEXT PRIMARY KEY,
    movie_id        TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    release_guid    TEXT NOT NULL,
    release_title   TEXT NOT NULL,
    indexer_id      TEXT,
    protocol        TEXT NOT NULL DEFAULT '',
    size            BIGINT NOT NULL DEFAULT 0,
    added_at        TIMESTAMPTZ NOT NULL,
    notes           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_blocklist_movie_id    ON blocklist(movie_id);
CREATE UNIQUE INDEX idx_blocklist_guid ON blocklist(release_guid);

-- ── Quality definitions ─────────────────────────────────────────────────────

CREATE TABLE quality_definitions (
    id             TEXT    PRIMARY KEY,
    name           TEXT    NOT NULL,
    resolution     TEXT    NOT NULL,
    source         TEXT    NOT NULL,
    codec          TEXT    NOT NULL,
    hdr            TEXT    NOT NULL,
    min_size       REAL    NOT NULL DEFAULT 0,
    max_size       REAL    NOT NULL DEFAULT 0,
    sort_order     INTEGER NOT NULL DEFAULT 0,
    preferred_size REAL    NOT NULL DEFAULT 0
);

INSERT INTO quality_definitions (id, name, resolution, source, codec, hdr, min_size, max_size, preferred_size, sort_order) VALUES
  ('2160p-remux-x265-hdr10',       '2160p Remux HDR',   '2160p',  'remux',  'x265', 'hdr10', 35, 800, 800, 10),
  ('2160p-bluray-x265-hdr10',      '2160p Bluray HDR',  '2160p',  'bluray', 'x265', 'hdr10', 15, 250, 250, 20),
  ('2160p-webdl-x265-hdr10',       '2160p WEBDL HDR',   '2160p',  'webdl',  'x265', 'hdr10', 15, 250, 250, 30),
  ('2160p-webrip-x265-hdr10',      '2160p WEBRip HDR',  '2160p',  'webrip', 'x265', 'hdr10', 15, 250, 250, 40),
  ('2160p-hdtv-x265-hdr10',        '2160p HDTV HDR',    '2160p',  'hdtv',   'x265', 'hdr10', 15, 250, 250, 50),
  ('1080p-remux-x265-none',        '1080p Remux',       '1080p',  'remux',  'x265', 'none',  17, 400, 400, 60),
  ('1080p-bluray-x265-none',       '1080p Bluray',      '1080p',  'bluray', 'x265', 'none',   4,  95,  95, 70),
  ('1080p-webdl-x264-none',        '1080p WEBDL',       '1080p',  'webdl',  'x264', 'none',   4,  40,  40, 80),
  ('1080p-webrip-x265-none',       '1080p WEBRip',      '1080p',  'webrip', 'x265', 'none',   4,  40,  40, 90),
  ('1080p-hdtv-x264-none',         '1080p HDTV',        '1080p',  'hdtv',   'x264', 'none',   4,  40,  40, 100),
  ('720p-bluray-x264-none',        '720p Bluray',       '720p',   'bluray', 'x264', 'none',   2,  30,  30, 110),
  ('720p-webdl-x264-none',         '720p WEBDL',        '720p',   'webdl',  'x264', 'none',   2,  20,  20, 120),
  ('720p-webrip-x264-none',        '720p WEBRip',       '720p',   'webrip', 'x264', 'none',   2,  20,  20, 130),
  ('720p-hdtv-x264-none',          '720p HDTV',         '720p',   'hdtv',   'x264', 'none',   2,  20,  20, 140),
  ('576p-bluray-x264-none',        '576p Bluray',       '576p',   'bluray', 'x264', 'none',   0,   5,   5, 150),
  ('480p-bluray-x264-none',        '480p Bluray',       '480p',   'bluray', 'x264', 'none',   0,   5,   5, 160),
  ('480p-webdl-x264-none',         '480p WEBDL',        '480p',   'webdl',  'x264', 'none',   0,   3,   3, 170),
  ('480p-webrip-x264-none',        '480p WEBRip',       '480p',   'webrip', 'x264', 'none',   0,   3,   3, 180),
  ('sd-dvdr-unknown-none',         'DVD-R',             'sd',     'dvdr',   'unknown', 'none',  0,  0,   0, 190),
  ('sd-dvd-xvid-none',             'SD DVD',            'sd',     'dvd',    'xvid', 'none',   0,   3,   3, 200),
  ('sd-hdtv-x264-none',            'SD HDTV',           'sd',     'hdtv',   'x264', 'none',   0,   3,   3, 210),
  ('unknown-brdisk-unknown-none',  'BR-DISK',           'unknown','brdisk', 'unknown', 'none', 0,  0,   0, 220),
  ('unknown-rawhd-unknown-none',   'Raw-HD',            'unknown','rawhd',  'unknown', 'none', 0,  0,   0, 230),
  ('sd-dvdscr-unknown-none',       'DVDSCR',            'sd',     'dvdscr', 'unknown', 'none', 0,  3,   0, 240),
  ('sd-regional-unknown-none',     'Regional',          'sd',     'regional','unknown','none',  0,  3,   0, 250),
  ('unknown-telecine-unknown-none','Telecine',           'unknown','telecine','unknown','none',  0,  0,   0, 260),
  ('unknown-telesync-unknown-none','Telesync',           'unknown','telesync','unknown','none',  0,  0,   0, 270),
  ('unknown-cam-unknown-none',     'CAM',               'unknown','cam',    'unknown', 'none',  0,  0,   0, 280),
  ('unknown-workprint-unknown-none','Workprint',         'unknown','workprint','unknown','none', 0,  0,   0, 290);

-- ── Media management ────────────────────────────────────────────────────────

CREATE TABLE media_management (
    id                       INTEGER PRIMARY KEY CHECK (id = 1),
    rename_movies            BOOLEAN NOT NULL DEFAULT TRUE,
    standard_movie_format    TEXT    NOT NULL DEFAULT '{Movie Title} ({Release Year}) {Quality Full}',
    movie_folder_format      TEXT    NOT NULL DEFAULT '{Movie Title} ({Release Year})',
    colon_replacement        TEXT    NOT NULL DEFAULT 'space-dash',
    import_extra_files       BOOLEAN NOT NULL DEFAULT FALSE,
    extra_file_extensions    TEXT    NOT NULL DEFAULT 'srt,nfo',
    unmonitor_deleted_movies BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO media_management (id) VALUES (1);

-- ── Download handling ───────────────────────────────────────────────────────

CREATE TABLE download_handling (
    id                            INTEGER PRIMARY KEY CHECK (id = 1),
    enable_completed              BOOLEAN NOT NULL DEFAULT TRUE,
    check_interval_minutes        INTEGER NOT NULL DEFAULT 1,
    redownload_failed             BOOLEAN NOT NULL DEFAULT TRUE,
    redownload_failed_interactive BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO download_handling (id) VALUES (1);

-- ── Remote path mappings ────────────────────────────────────────────────────

CREATE TABLE remote_path_mappings (
    id          TEXT NOT NULL PRIMARY KEY,
    host        TEXT NOT NULL,
    remote_path TEXT NOT NULL,
    local_path  TEXT NOT NULL
);

-- ── Library file candidates ─────────────────────────────────────────────────

CREATE TABLE library_file_candidates (
    library_id           TEXT    NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    file_path            TEXT    NOT NULL,
    file_size            BIGINT NOT NULL DEFAULT 0,
    parsed_title         TEXT    NOT NULL DEFAULT '',
    parsed_year          INTEGER NOT NULL DEFAULT 0,
    tmdb_id              INTEGER NOT NULL DEFAULT 0,
    tmdb_title           TEXT    NOT NULL DEFAULT '',
    tmdb_year            INTEGER NOT NULL DEFAULT 0,
    tmdb_original_title  TEXT    NOT NULL DEFAULT '',
    auto_matched         BOOLEAN NOT NULL DEFAULT FALSE,
    scanned_at           TEXT    NOT NULL,
    matched_at           TEXT,
    PRIMARY KEY (library_id, file_path)
);

-- ── Storage snapshots ───────────────────────────────────────────────────────

CREATE TABLE storage_snapshots (
    id           TEXT PRIMARY KEY,
    captured_at  TIMESTAMPTZ NOT NULL,
    total_bytes  BIGINT NOT NULL DEFAULT 0,
    file_count   BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_storage_snapshots_captured_at ON storage_snapshots(captured_at);

-- ── Collections ─────────────────────────────────────────────────────────────

CREATE TABLE collections (
    id              TEXT    PRIMARY KEY,
    name            TEXT    NOT NULL,
    person_id       BIGINT NOT NULL,
    person_type     TEXT    NOT NULL DEFAULT 'director',
    created_at      TIMESTAMPTZ NOT NULL,
    total_items     INTEGER NOT NULL DEFAULT 0,
    in_library_items INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_collections_person ON collections(person_id, person_type);

-- ── Media server configs ────────────────────────────────────────────────────

CREATE TABLE media_server_configs (
    id         TEXT PRIMARY KEY,
    name       TEXT    NOT NULL,
    kind       TEXT    NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    settings   TEXT    NOT NULL DEFAULT '{}',
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL
);

-- ── Tags ────────────────────────────────────────────────────────────────────

CREATE TABLE tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE movie_tags (
    movie_id TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    tag_id   TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (movie_id, tag_id)
);

CREATE TABLE indexer_tags (
    indexer_id TEXT NOT NULL REFERENCES indexer_configs(id) ON DELETE CASCADE,
    tag_id     TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (indexer_id, tag_id)
);

CREATE TABLE download_client_tags (
    download_client_id TEXT NOT NULL REFERENCES download_client_configs(id) ON DELETE CASCADE,
    tag_id             TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (download_client_id, tag_id)
);

CREATE TABLE notification_tags (
    notification_id TEXT NOT NULL REFERENCES notification_configs(id) ON DELETE CASCADE,
    tag_id          TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (notification_id, tag_id)
);

-- ── Custom formats ──────────────────────────────────────────────────────────

CREATE TABLE custom_formats (
    id                      TEXT PRIMARY KEY,
    name                    TEXT NOT NULL UNIQUE,
    include_when_renaming   BOOLEAN NOT NULL DEFAULT FALSE,
    specifications_json     TEXT NOT NULL DEFAULT '[]',
    created_at              TEXT NOT NULL,
    updated_at              TEXT NOT NULL
);

CREATE TABLE custom_format_scores (
    quality_profile_id TEXT NOT NULL REFERENCES quality_profiles(id) ON DELETE CASCADE,
    custom_format_id   TEXT NOT NULL REFERENCES custom_formats(id) ON DELETE CASCADE,
    score              INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (quality_profile_id, custom_format_id)
);

-- ── Import lists ────────────────────────────────────────────────────────────

CREATE TABLE import_list_configs (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    kind                TEXT NOT NULL,
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    settings            TEXT NOT NULL DEFAULT '{}',
    search_on_add       BOOLEAN NOT NULL DEFAULT FALSE,
    monitor             BOOLEAN NOT NULL DEFAULT TRUE,
    min_availability    TEXT NOT NULL DEFAULT 'released',
    quality_profile_id  TEXT NOT NULL DEFAULT '',
    library_id          TEXT NOT NULL DEFAULT '',
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL
);

CREATE TABLE import_exclusions (
    id         TEXT PRIMARY KEY,
    tmdb_id    INTEGER NOT NULL UNIQUE,
    title      TEXT NOT NULL DEFAULT '',
    year       INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE TABLE import_list_tags (
    import_list_id TEXT NOT NULL REFERENCES import_list_configs(id) ON DELETE CASCADE,
    tag_id         TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (import_list_id, tag_id)
);

-- ── Activity log ────────────────────────────────────────────────────────────

CREATE TABLE activity_log (
    id         TEXT PRIMARY KEY,
    type       TEXT NOT NULL,
    category   TEXT NOT NULL,
    movie_id   TEXT,
    title      TEXT NOT NULL,
    detail     TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_activity_log_created  ON activity_log(created_at DESC);
CREATE INDEX idx_activity_log_category ON activity_log(category);
CREATE INDEX idx_activity_log_movie    ON activity_log(movie_id);

-- ── Watch history ───────────────────────────────────────────────────────────

CREATE TABLE watch_history (
    id         TEXT PRIMARY KEY,
    movie_id   TEXT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    tmdb_id    INTEGER NOT NULL,
    watched_at TEXT NOT NULL,
    user_name  TEXT NOT NULL DEFAULT '',
    source     TEXT NOT NULL,
    UNIQUE(movie_id, watched_at, user_name)
);

CREATE INDEX idx_watch_history_movie   ON watch_history(movie_id);
CREATE INDEX idx_watch_history_watched ON watch_history(watched_at DESC);

CREATE TABLE watch_sync_state (
    media_server_id TEXT PRIMARY KEY,
    last_sync_at    TEXT NOT NULL
);

-- +goose Down

DROP TABLE IF EXISTS watch_sync_state;
DROP TABLE IF EXISTS watch_history;
DROP TABLE IF EXISTS activity_log;
DROP TABLE IF EXISTS import_list_tags;
DROP TABLE IF EXISTS import_exclusions;
DROP TABLE IF EXISTS import_list_configs;
DROP TABLE IF EXISTS custom_format_scores;
DROP TABLE IF EXISTS custom_formats;
DROP TABLE IF EXISTS notification_tags;
DROP TABLE IF EXISTS download_client_tags;
DROP TABLE IF EXISTS indexer_tags;
DROP TABLE IF EXISTS movie_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS media_server_configs;
DROP TABLE IF EXISTS collections;
DROP TABLE IF EXISTS storage_snapshots;
DROP TABLE IF EXISTS library_file_candidates;
DROP TABLE IF EXISTS remote_path_mappings;
DROP TABLE IF EXISTS download_handling;
DROP TABLE IF EXISTS media_management;
DROP TABLE IF EXISTS quality_definitions;
DROP TABLE IF EXISTS blocklist;
DROP TABLE IF EXISTS notification_configs;
DROP TABLE IF EXISTS download_client_configs;
DROP TABLE IF EXISTS grab_history;
DROP TABLE IF EXISTS indexer_configs;
DROP TABLE IF EXISTS movie_files;
DROP TABLE IF EXISTS movies;
DROP TABLE IF EXISTS libraries;
DROP TABLE IF EXISTS quality_profiles;
