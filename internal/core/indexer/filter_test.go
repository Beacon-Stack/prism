package indexer

// filter_test.go — Regression suite for the per-indexer min_seeders
// filter. Mirrors Pilot's coverage. The filter must be:
//
//   - per-indexer (each indexer gets its own threshold from the DB)
//   - tagging, not dropping (UI keeps a hand on the steering wheel)
//   - freshness-aware (recent releases confirmed on 2+ indexers bypass)
//   - tolerant of missing entries (default 5 when no threshold set)
//
// If any of these regress, the manual-search dialog will either hide
// content or surface dead torrents — both bad. Pilot's CLAUDE.md has
// the same suite documented.

import (
	"strings"
	"testing"

	"github.com/beacon-stack/prism/pkg/plugin"
)

func mkResult(title, indexerID string, seeds int, ageDays float64) SearchResult {
	return SearchResult{
		Release: plugin.Release{
			Title:   title,
			Indexer: indexerID, // not strictly the name, but unique enough for tests
			Seeds:   seeds,
			AgeDays: ageDays,
		},
		IndexerID: indexerID,
	}
}

func TestApplyMinSeedersFilter_BelowThresholdGetsTagged(t *testing.T) {
	results := []SearchResult{
		mkResult("Movie 2023", "tgc", 3, 5),  // below 5 → tagged
		mkResult("Movie 2023", "1337x", 7, 5), // above 5 → clean
	}
	applyMinSeedersFilter(results, map[string]int{"tgc": 5, "1337x": 5})

	if len(results[0].FilterReasons) == 0 {
		t.Errorf("expected tgc result with 3 seeds to be tagged below threshold")
	} else if !strings.Contains(results[0].FilterReasons[0], "below minimum seeders") {
		t.Errorf("unexpected reason: %q", results[0].FilterReasons[0])
	}
	if len(results[1].FilterReasons) != 0 {
		t.Errorf("1337x with 7 seeds should NOT be tagged; got %v", results[1].FilterReasons)
	}
}

func TestApplyMinSeedersFilter_AboveThresholdUntagged(t *testing.T) {
	results := []SearchResult{
		mkResult("Movie", "x", 100, 30),
		mkResult("Movie", "x", 6, 30),
	}
	applyMinSeedersFilter(results, map[string]int{"x": 5})
	for i, r := range results {
		if len(r.FilterReasons) != 0 {
			t.Errorf("result[%d] seeds=%d should not be tagged; got %v", i, r.Seeds, r.FilterReasons)
		}
	}
}

// Freshness exception: AgeDays < 0.5 AND 2+ distinct indexers reporting
// the same title bypass the filter. Brand-new uploads legitimately
// have low/zero seed counts on their first scrape.
func TestApplyMinSeedersFilter_FreshMultiIndexerBypass(t *testing.T) {
	results := []SearchResult{
		mkResult("Hot Release 2026", "indexerA", 0, 0.1), // 2.4 hours old, 0 seeds
		mkResult("Hot Release 2026", "indexerB", 1, 0.1),
	}
	applyMinSeedersFilter(results, map[string]int{"indexerA": 5, "indexerB": 5})

	for i, r := range results {
		if len(r.FilterReasons) != 0 {
			t.Errorf("result[%d] should bypass filter (fresh, 2+ indexers); got %v", i, r.FilterReasons)
		}
	}
}

func TestApplyMinSeedersFilter_OldReleaseNoBypass(t *testing.T) {
	// Old release with low seeds, even on multiple indexers, gets tagged.
	// "Old" in this context = >= 12h.
	results := []SearchResult{
		mkResult("Stale Movie", "a", 0, 30),
		mkResult("Stale Movie", "b", 0, 30),
	}
	applyMinSeedersFilter(results, map[string]int{"a": 5, "b": 5})
	for i, r := range results {
		if len(r.FilterReasons) == 0 {
			t.Errorf("result[%d] is old (30d) and dead (0 seeds) — must be tagged", i)
		}
	}
}

func TestApplyMinSeedersFilter_FreshSingleIndexerNoBypass(t *testing.T) {
	// Fresh upload but only ONE indexer reporting it — no second
	// confirmation, so the filter still applies. Guards against a single
	// bad indexer claiming a release is fresh when it isn't.
	results := []SearchResult{
		mkResult("Untrusted Solo Release", "indexerA", 0, 0.1),
	}
	applyMinSeedersFilter(results, map[string]int{"indexerA": 5})

	if len(results[0].FilterReasons) == 0 {
		t.Errorf("solo fresh result with 0 seeds must still be tagged; second-indexer confirmation missing")
	}
}

func TestApplyMinSeedersFilter_DefaultWhenIndexerMissing(t *testing.T) {
	// An indexer ID that isn't in the threshold map falls back to 5.
	results := []SearchResult{
		mkResult("Unknown Indexer Movie", "ghost", 4, 30), // below default 5
	}
	applyMinSeedersFilter(results, map[string]int{}) // empty map

	if len(results[0].FilterReasons) == 0 {
		t.Errorf("missing indexer should fall back to default=5; result with 4 seeds must be tagged")
	}
}

func TestApplyMinSeedersFilter_DefaultWhenIndexerZero(t *testing.T) {
	// A zero/missing entry in the threshold map also falls back to default.
	results := []SearchResult{
		mkResult("Zero Configured", "wonky", 4, 30),
	}
	applyMinSeedersFilter(results, map[string]int{"wonky": 0}) // zero is treated as "default please"

	if len(results[0].FilterReasons) == 0 {
		t.Errorf("threshold of 0 must fall back to default=5, not literally 0")
	}
}

func TestApplyMinSeedersFilter_PerIndexerDifferentThresholds(t *testing.T) {
	// Public indexers can have a higher threshold (say 10) while a
	// private tracker keeps a low one (say 1). 5 seeds passes private
	// but fails public.
	results := []SearchResult{
		mkResult("X", "public", 5, 30),  // < 10 → tagged
		mkResult("Y", "private", 5, 30), // ≥ 1 → clean
	}
	applyMinSeedersFilter(results, map[string]int{"public": 10, "private": 1})

	if len(results[0].FilterReasons) == 0 {
		t.Errorf("public with 5 seeds must be tagged (threshold 10)")
	}
	if len(results[1].FilterReasons) != 0 {
		t.Errorf("private with 5 seeds must be clean (threshold 1); got %v", results[1].FilterReasons)
	}
}

// Headline regression: an indexer that lies about a 5-year-old dead
// torrent ("847 seeders!") still gets tagged, but its title alone won't
// trigger the freshness bypass because AgeDays > 0.5.
func TestApplyMinSeedersFilter_DoesNotBypassOnInflatedSeeders(t *testing.T) {
	// Inflated count BUT old release. The 800+ seeders happen to clear
	// the threshold (above 5), so nothing should be tagged in this case
	// — the test is documentation that high reported seed counts are
	// the indexer-trust problem, not the min-seeders filter's problem.
	// (A different mechanism — the dedup median — handles inflation.)
	results := []SearchResult{
		mkResult("Old Inflated", "shadypublic", 847, 1825), // 5y old, "847 seeds"
	}
	applyMinSeedersFilter(results, map[string]int{"shadypublic": 5})

	if len(results[0].FilterReasons) != 0 {
		t.Errorf("847 ≥ 5: should NOT be tagged by min-seeders filter; got %v",
			results[0].FilterReasons)
	}
}
