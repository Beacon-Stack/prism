-- +goose Up
CREATE TABLE quality_definitions (
    id         TEXT    PRIMARY KEY,  -- stable slug: "<resolution>-<source>-<codec>-<hdr>"
    name       TEXT    NOT NULL,     -- human-readable label, e.g. "1080p Bluray"
    resolution TEXT    NOT NULL,
    source     TEXT    NOT NULL,
    codec      TEXT    NOT NULL,
    hdr        TEXT    NOT NULL,
    min_size   REAL    NOT NULL DEFAULT 0,   -- MB per minute of runtime (0 = no minimum)
    max_size   REAL    NOT NULL DEFAULT 0,   -- MB per minute of runtime (0 = no limit)
    sort_order INTEGER NOT NULL DEFAULT 0
);

-- Seed with the 14 standard quality levels (lowest → highest).
INSERT INTO quality_definitions (id, name, resolution, source, codec, hdr, min_size, max_size, sort_order) VALUES
  ('sd-dvd-xvid-none',       'SD DVD',          'sd',    'dvd',    'xvid', 'none',  0,    3,   10),
  ('sd-hdtv-x264-none',      'SD HDTV',         'sd',    'hdtv',   'x264', 'none',  0,    3,   20),
  ('720p-hdtv-x264-none',    '720p HDTV',       '720p',  'hdtv',   'x264', 'none',  2,   20,   30),
  ('720p-webdl-x264-none',   '720p WEBDL',      '720p',  'webdl',  'x264', 'none',  2,   20,   40),
  ('720p-webrip-x264-none',  '720p WEBRip',     '720p',  'webrip', 'x264', 'none',  2,   20,   50),
  ('720p-bluray-x264-none',  '720p Bluray',     '720p',  'bluray', 'x264', 'none',  2,   30,   60),
  ('1080p-hdtv-x264-none',   '1080p HDTV',      '1080p', 'hdtv',   'x264', 'none',  4,   40,   70),
  ('1080p-webdl-x264-none',  '1080p WEBDL',     '1080p', 'webdl',  'x264', 'none',  4,   40,   80),
  ('1080p-webrip-x265-none', '1080p WEBRip',    '1080p', 'webrip', 'x265', 'none',  4,   40,   90),
  ('1080p-bluray-x265-none', '1080p Bluray',    '1080p', 'bluray', 'x265', 'none',  4,   95,  100),
  ('1080p-remux-x265-none',  '1080p Remux',     '1080p', 'remux',  'x265', 'none', 17,  400,  110),
  ('2160p-webdl-x265-hdr10', '2160p WEBDL HDR', '2160p', 'webdl',  'x265', 'hdr10', 15, 250,  120),
  ('2160p-bluray-x265-hdr10','2160p Bluray HDR','2160p', 'bluray', 'x265', 'hdr10', 15, 250,  130),
  ('2160p-remux-x265-hdr10', '2160p Remux HDR', '2160p', 'remux',  'x265', 'hdr10', 35, 800,  140);

-- +goose Down
DROP TABLE quality_definitions;
