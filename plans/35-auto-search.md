# Plan 35 — On-Demand Automatic Search ✅ COMPLETE

Implemented in v0.5.3.

## Problem

Radarr has three search modes:
1. **RSS Sync** — background job polls indexers for recent releases, auto-grabs matches (we have this)
2. **Interactive/Manual Search** — user browses releases and picks one (we have this)
3. **Automatic Search** — user clicks a button, Radarr does a full search across all indexers, picks the best release that satisfies the quality profile, and sends it to the download client automatically (we DON'T have this)

Mode 3 is the most common way Radarr users trigger downloads. It's the "just get me this movie" button.

## Design

### Service Layer — `internal/core/autosearch/service.go`

The auto-search orchestration logic lives in a **new service**, not in the API handler layer. This is the single source of truth for "search + filter + grab" — called by the API handler, and eventually by RSS sync too (to eliminate the current drift between rss_sync.go and releases.go).

```go
type Service struct {
    indexerSvc    *indexer.Service
    movieSvc      *movie.Service
    downloaderSvc *downloader.Service
    blocklistSvc  *blocklist.Service
    qualitySvc    *quality.Service
    bus           *events.Bus
    logger        *slog.Logger
}

// Result describes the outcome of an auto-search for a single movie.
type Result struct {
    Status string          // "grabbed", "no_match", "skipped"
    Reason string          // human-readable reason (empty on success)
    Grab   *GrabRecord     // populated only when Status == "grabbed"
}

// SearchMovie performs a full indexer search for a single movie, picks the
// best release satisfying the quality profile, and submits it to a download
// client. Works on both monitored and unmonitored movies (explicit user action).
func (s *Service) SearchMovie(ctx context.Context, movieID string) (*Result, error)

// SearchMovies runs SearchMovie for each movie ID with concurrency control.
// Max 2 concurrent searches, 1s delay between starts. movie_ids capped at 100.
func (s *Service) SearchMovies(ctx context.Context, movieIDs []string) (*BulkResult, error)
```

**`SearchMovie` logic:**
1. Fetch movie via `movieSvc.Get()`
2. Check for active grab (skip with `"skipped"` / `"already downloading"` if exists)
3. Full indexer search via `indexerSvc.Search()`
4. Load quality profile via `qualitySvc.Get()`
5. Get current file quality via `movieSvc.ListFiles()` → `bestFileQuality()`
6. Iterate candidates (sorted best→worst by quality score, then seeds):
   a. Skip if blocklisted (`blocklistSvc.IsBlocklisted()`)
   b. Skip if `!prof.WantRelease()`
   c. Try `downloaderSvc.Add()` — on failure, auto-blocklist and try next candidate
   d. On success: record grab via `indexerSvc.Grab()`, return `"grabbed"`
7. If all candidates exhausted → return `"no_match"`

Key difference from the plan v1: **iterate candidates with fallback**, don't just pick the top one. If the download client rejects a release, auto-blocklist it and try the next.

**`SearchMovies` bulk logic:**
- Validates `len(movieIDs) <= 100`, rejects with error if exceeded
- Processes movies with bounded concurrency (2 goroutines, 1s stagger between starts)
- Collects per-movie results into `BulkResult`
- Fires a single summary notification event (not per-movie)

### API

**`POST /api/v1/movies/{id}/search`** — single-movie automatic search

Works on both monitored and unmonitored movies (following Radarr — this is an explicit user action).

Response uses structured body with a `result` field so the frontend can programmatically distinguish outcomes:

| Scenario | Status | Body |
|----------|--------|------|
| Grabbed successfully | 200 | `{ "result": "grabbed", "grab": {GrabHistory} }` |
| No suitable release | 200 | `{ "result": "no_match", "reason": "no releases satisfy quality profile" }` |
| Already downloading | 200 | `{ "result": "skipped", "reason": "already downloading" }` |
| Movie not found | 404 | huma error |
| All indexers failed | 502 | huma error |
| No download client | 503 | huma error |

**`POST /api/v1/movies/search`** — bulk automatic search

- Body: `{ "movie_ids": ["uuid1", "uuid2", ...] }` — max 100
- Returns 202 Accepted immediately, processes async
- Pushes progress over WebSocket (we already have `ws.Hub`)
- Final WS message: `{ "type": "bulk_search_complete", "searched": 5, "grabbed": 3, "results": [...] }`
- Also fires a single summary notification event for Discord/webhook subscribers

WS progress messages during processing:
```json
{ "type": "bulk_search_progress", "movie_id": "uuid1", "result": "grabbed", "current": 1, "total": 5 }
{ "type": "bulk_search_progress", "movie_id": "uuid2", "result": "no_match", "current": 2, "total": 5 }
...
{ "type": "bulk_search_complete", "searched": 5, "grabbed": 3, "results": [...] }
```

### Race Condition Mitigation — Active Grab Dedup

The TOCTOU gap between checking `ListActiveGrabs` and inserting a new grab can cause duplicate downloads (auto-search + RSS sync running simultaneously, or two browser tabs).

**Fix:** Add a unique partial index on `grab_history`:

```sql
CREATE UNIQUE INDEX idx_grab_history_active_movie
ON grab_history (movie_id)
WHERE download_status NOT IN ('completed', 'failed', 'removed');
```

Then `indexerSvc.Grab()` returns a distinct error on conflict, and `SearchMovie` returns `"skipped"` / `"already downloading"`. This also fixes the same latent bug in RSS sync.

### Shared Helpers

Move out of `rss_sync.go` into `internal/core/autosearch/`:
- `bestFileQuality(files []movie.FileInfo) plugin.Quality` — adapted to use `movie.FileInfo` instead of raw DB rows
- `movieEligibleForGrab()` stays in rss_sync.go (only RSS sync uses min-availability filtering; auto-search skips it since it's an explicit user action)

### Notifications

Bulk search fires a **single summary event** (not per-movie) so users don't get 50 Discord notifications:

```go
events.TypeBulkSearchComplete → { Searched: 5, Grabbed: 3, Results: [...] }
```

Single-movie auto-search fires the existing `TypeGrabStarted` event on success (same as manual grab).

### Dependencies (all exist)

| Need | Source |
|------|--------|
| Full indexer search | `indexerSvc.Search(ctx, query)` |
| Quality profile | `qualitySvc.Get(ctx, profileID)` |
| Current file quality | `movieSvc.ListFiles(ctx, movieID)` — no raw Querier needed |
| `WantRelease` filtering | `prof.WantRelease(releaseQuality, currentQuality)` |
| Blocklist check + add | `blocklistSvc.IsBlocklisted()`, `blocklistSvc.Add()` |
| Submit to downloader | `downloaderSvc.Add(ctx, release)` |
| Record grab history | `indexerSvc.Grab(ctx, ...)` |
| Active grab dedup | DB unique partial index (new migration) |
| WebSocket push | `ws.Hub.Broadcast()` |

### Frontend

**MovieDetail page** — add auto-search button next to existing "Search" button:
- "Search" (existing) → opens ManualSearchModal (interactive)
- New button (e.g., magnifying glass icon with auto indicator) → calls `POST /movies/{id}/search`
- Shows toast on success ("Grabbed: Release.Title"), "no_match" ("No suitable release found"), or "skipped" ("Already downloading")
- Loading/spinner state while search runs

**Wanted page** — add "Search All Missing" / "Search All Cutoff Unmet" buttons:
- Calls bulk endpoint, gets 202 back
- Subscribes to WS progress messages, shows progress bar / live counter
- Final toast with summary ("Grabbed 3 of 5 movies")

**Bulk Movie Editor** — add "Search Selected" action:
- Same bulk endpoint with selected movie IDs

### Files to Change

**Backend:**
| File | Change |
|------|--------|
| `internal/core/autosearch/service.go` | **New.** Core `SearchMovie` + `SearchMovies` logic |
| `internal/core/autosearch/service_test.go` | **New.** Unit tests with mocked dependencies |
| `internal/api/v1/releases.go` | Add `POST /movies/{id}/search` + `POST /movies/search` endpoints (thin handlers calling autosearch service) |
| `internal/api/router.go` | Create `autosearch.Service`, pass to `RegisterReleaseRoutes` |
| `internal/db/migrations/NNNN_grab_active_unique.sql` | **New.** Unique partial index on grab_history |
| `internal/events/types.go` | Add `TypeBulkSearchComplete` event type |

**Frontend:**
| File | Change |
|------|--------|
| `web/ui/src/api/movies.ts` | Add `useAutoSearch(movieId)` and `useBulkAutoSearch()` hooks |
| `web/ui/src/types/index.ts` | Add `AutoSearchResult`, `BulkSearchResult` types |
| `web/ui/src/pages/movies/MovieDetail.tsx` | Add auto-search button next to manual search button |
| `web/ui/src/pages/wanted/WantedPage.tsx` | Add "Search All" buttons to Missing and Cutoff tabs |

### Build Order

1. DB migration: unique partial index on `grab_history`
2. `internal/core/autosearch/service.go` — `SearchMovie` method
3. `POST /api/v1/movies/{id}/search` endpoint (thin handler)
4. Tests for `SearchMovie`
5. `SearchMovies` bulk method with concurrency control
6. `POST /api/v1/movies/search` endpoint (202 + async + WS progress)
7. `TypeBulkSearchComplete` event + summary notification
8. Tests for `SearchMovies`
9. Frontend: API hooks + types
10. Frontend: MovieDetail auto-search button
11. Frontend: Wanted page bulk search buttons

### Test Cases

**Single auto-search (`SearchMovie`):**
- Movie with no file on disk → grabs best release matching profile
- Movie with file below cutoff → grabs upgrade
- Movie at/above cutoff, upgrades disabled → returns `no_match`
- All releases blocklisted → returns `no_match`
- No indexers configured → returns `no_match` (empty results)
- Movie not found → error (404 at API layer)
- No download client → error (503 at API layer)
- Active grab already exists (unique index conflict) → returns `skipped`
- Top release rejected by download client → auto-blocklists, tries next candidate
- Unmonitored movie → still works (explicit user action)

**Bulk auto-search (`SearchMovies`):**
- Mix of grabbable and non-grabbable movies → returns correct per-movie results
- Empty movie_ids → returns `{ searched: 0, grabbed: 0, results: [] }`
- movie_ids > 100 → rejected with error
- Invalid movie ID in list → counted as failed, doesn't abort others
- Concurrency: only 2 searches run at a time
- WS messages sent for each movie as it completes

**Race condition:**
- Two concurrent `SearchMovie` calls for same movie → one succeeds, other gets `skipped` (unique index)
- `SearchMovie` + RSS sync for same movie → same dedup behavior
