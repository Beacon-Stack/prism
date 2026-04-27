package indexer

import "fmt"

// applyMinSeedersFilter tags each result whose seed count falls below the
// configured per-indexer minimum with a FilterReasons string. Results are
// NOT dropped — the manual-search UI renders flagged rows grayed with an
// "override and grab anyway" button so users can recover false positives.
// Hiding rows creates content-loss traps and frustrated users; tagging
// preserves agency.
//
// Freshness exception: releases less than 12 hours old AND confirmed on
// 2+ distinct indexers bypass the filter. Rationale: public-tracker
// aggregators lag behind real seeder counts by minutes-to-hours on
// brand-new releases, so a just-uploaded popular movie may legitimately
// show 0 seeders on its first indexer hit. Requiring a second-indexer
// confirmation guards against a single bad source claiming a release is
// fresh when it's not.
//
// minByIndexer is the per-indexer threshold map keyed by indexer ID.
// Missing/zero entries fall back to a sensible default of 5 — matters
// for edge cases where an indexer exists in search results but not in
// the rows list (shouldn't happen in production, but tests may
// construct such cases).
//
// Mirrors pilot/internal/core/indexer/service.go:applyMinSeedersFilter
// — keep the two in sync; the parameters are tuned to the same
// public-tracker behavior. Pilot has a comprehensive regression suite
// for this logic; the Prism suite here covers the same ground.
func applyMinSeedersFilter(results []SearchResult, minByIndexer map[string]int) {
	titleIndexers := make(map[string]map[string]bool)
	for _, r := range results {
		if titleIndexers[r.Title] == nil {
			titleIndexers[r.Title] = make(map[string]bool)
		}
		titleIndexers[r.Title][r.IndexerID] = true
	}

	for i := range results {
		r := &results[i]
		minSeeds := minByIndexer[r.IndexerID]
		if minSeeds <= 0 {
			minSeeds = 5
		}
		if r.Seeds < minSeeds {
			fresh := r.AgeDays < 0.5 && len(titleIndexers[r.Title]) >= 2
			if !fresh {
				r.FilterReasons = append(r.FilterReasons,
					fmt.Sprintf("below minimum seeders (%d < %d)", r.Seeds, minSeeds))
			}
		}
	}
}
