-- +goose Up

-- Per-indexer minimum-seeders threshold. Releases below this floor are
-- *tagged* with a filter reason (NOT dropped) so the manual-search UI
-- can render them grayed-with-override; auto-search ignores tagged rows.
--
-- Default 5 mirrors Pilot's existing default — chosen because public
-- trackers' lowest seed bands tend to be dead/dying torrents, and the
-- 0–4 band catches the worst pollution without hiding legitimate
-- low-seed releases. TRaSH Guides recommends 20+ for public trackers;
-- users can bump per-indexer via the settings UI.
--
-- The freshness exception (releases < 12h old + confirmed on 2+
-- indexers bypass the filter) handles the case where a brand-new
-- upload legitimately has 0 seeders for a few minutes before peers
-- catch up. See applyMinSeedersFilter in internal/core/indexer/.
ALTER TABLE indexer_configs
    ADD COLUMN min_seeders INTEGER NOT NULL DEFAULT 5;

-- +goose Down

ALTER TABLE indexer_configs DROP COLUMN IF EXISTS min_seeders;
