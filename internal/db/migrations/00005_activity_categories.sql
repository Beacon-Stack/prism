-- +goose Up

-- Tighten activity_log.category from a free-form TEXT to a small set of
-- explicit, success/failure-split values. Mirrors Pilot's
-- 00009_activity_categories.sql.
--
-- Without the split, the Activity-page "Needs attention" rail can't tell
-- a successful grab apart from a failed grab without re-parsing the
-- type column — a fragile contract that breaks the moment an emit site
-- introduces a new type name.
--
-- New vocabulary (kept in sync with internal/core/activity/service.go):
--
--   grab_succeeded     — TypeGrabStarted, TypeBulkSearchComplete
--   grab_failed        — TypeGrabFailed
--   import_succeeded   — TypeImportComplete, TypeDownloadDone
--   import_failed      — TypeImportFailed
--   task               — TypeTaskStarted / TypeTaskFinished (reserved)
--   health             — TypeHealthIssue / TypeHealthOK (reserved)
--   movie              — TypeMovieAdded / TypeMovieDeleted / TypeMovieUpdated
--
-- Existing rows: Prism only ever wrote `grab`, `import`, and `movie`.
-- The grab and import buckets are split by the `type` column, which
-- records the originating event name verbatim — a clean, lossless
-- rewrite. Legacy values stay accepted by the API validator so external
-- clients filtering on the old names don't break.

UPDATE activity_log SET category = 'grab_succeeded'
 WHERE category = 'grab' AND type IN ('grab_started', 'bulk_search_complete');
UPDATE activity_log SET category = 'grab_failed'
 WHERE category = 'grab' AND type = 'grab_failed';

UPDATE activity_log SET category = 'import_succeeded'
 WHERE category = 'import' AND type IN ('import_complete', 'download_done');
UPDATE activity_log SET category = 'import_failed'
 WHERE category = 'import' AND type = 'import_failed';

-- +goose Down

UPDATE activity_log SET category = 'grab'
 WHERE category IN ('grab_succeeded', 'grab_failed');
UPDATE activity_log SET category = 'import'
 WHERE category IN ('import_succeeded', 'import_failed');
