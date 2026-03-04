# Phase 19 — Radarr Parity Phase 2

Five features ordered by effort (smallest to largest).
Each section includes exact files to touch and SQL/code to write.

---

## Status: COMPLETE ✓

All five features confirmed implemented as of 2026-03-04.

## Implementation Order

1. **Queue: Blocklist & Remove** ✓ — `POST /api/v1/queue/{id}/blocklist` in `internal/api/v1/queue.go`
2. **History Filtering** ✓ — `download_status` + `protocol` query params on `GET /api/v1/history`; frontend filter bar in History page
3. **Bulk Movie Editor** ✓ — `BulkEditModal` in `src/pages/dashboard/Dashboard.tsx`
4. **Backup / Restore** ✓ — `GET /api/v1/system/backup` + `POST /api/v1/system/restore` in `internal/api/v1/backup.go`; `BackupSection` in `SystemPage.tsx`
5. **Interactive Import UI** ✓ — disk-scan + import modal in `src/pages/settings/libraries/LibraryList.tsx`

---

## 1. Queue: Blocklist & Remove

### What it does
User clicks "Blocklist & Remove" on a queue item. Luminarr:
1. Removes the download from the client (existing `RemoveFromQueue` logic)
2. Adds the release to the blocklist so it won't be grabbed again
3. Returns 204 — frontend then opens the movie's ManualSearchModal so user can pick a different release

This does NOT auto-trigger a new search — the manual search modal handles that.

### Backend

**No new service needed.** Orchestrate at handler level in `internal/api/v1/queue.go`:

Add a new endpoint alongside the existing DELETE:

```
POST /api/v1/queue/{id}/blocklist
```

Handler logic (in the existing `RegisterQueueRoutes`):
1. Load grab from `ListActiveGrabs`, find by ID
2. If has `DownloadClientID` + `ClientItemID`: call `client.Remove(ctx, clientItemID, false)`
3. Call `s.q.MarkGrabRemoved(ctx, grabID)`
4. Call `blocklistSvc.Add(ctx, grab.MovieID, grab.ReleaseGUID(?), grab.ReleaseTitle, grab.IndexerID, grab.Protocol, grab.Size, "blocklisted from queue")`

**Problem:** `grab_history` has the release title and indexer_id but does not store the release GUID.
The blocklist uses release_guid as a unique key to prevent re-grabs.

**Fix:** The grab_history already stores `release_title`. For blocklist purposes, use
`release_title` as the de-duplication key rather than GUID (already added in migration
00008_blocklist.sql as `UNIQUE INDEX idx_blocklist_guid ON blocklist(release_guid)`).

**Alternative (simpler, no schema change):** Use the grab history ID itself as the "guid" when
adding to blocklist. The actual de-duplication in the grab pipeline uses `release_guid` from
the indexer search results, which we don't store. So blocklisting from queue prevents by title
match. Update `IsBlocklisted` to also check by title:

```sql
-- name: IsBlocklistedByTitle :one
SELECT COUNT(*) FROM blocklist WHERE release_title = ?;
```

Add this to `internal/db/queries/sqlite/blocklist.sql`, run `sqlc generate`.

**Handler requires injected services:** `RegisterQueueRoutes` currently takes only `*queue.Service`.
Change signature to also accept `*blocklist.Service`.

Update `internal/api/router.go` to pass `cfg.BlocklistService` there.

### Frontend

In `web/ui/src/pages/queue/Queue.tsx`:
- Add a "Blocklist" button (or icon) next to the existing Remove button on each queue row
- On click: call `POST /queue/{id}/blocklist`
- On success: show toast "Blocklisted — use manual search to try another release"
- Navigate to `/movies/{movieId}` with tab=releases pre-selected (or just toast)

Add hook in `web/ui/src/api/queue.ts`:
```ts
export function useBlocklistQueueItem() {
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/queue/${id}/blocklist`, { method: "POST" }),
  });
}
```

### Tests

**Integration test** (`internal/api/v1/queue_blocklist_test.go`):
- `TestBlocklistQueueItem` — seed a grab_history record with active status, POST to endpoint,
  assert 204, assert grab_history.download_status = "removed", assert blocklist has 1 entry

---

## 2. History Filtering

### What it does
`GET /api/v1/history` gains optional query params:
- `download_status` — filter by status string (e.g. "completed", "failed")
- `protocol` — filter by protocol ("torrent", "nzb")

Filtering is done in Go after fetching (avoids dynamic SQL with sqlc).

### Backend

In `internal/api/v1/history.go`, extend `historyListInput`:

```go
type historyListInput struct {
    Limit          int    `query:"limit"           default:"100" minimum:"1" maximum:"1000"`
    DownloadStatus string `query:"download_status" doc:"Filter by status: completed, failed, queued, etc."`
    Protocol       string `query:"protocol"        doc:"Filter by protocol: torrent, nzb"`
}
```

In handler, after fetching all rows, filter:
```go
if input.DownloadStatus != "" {
    rows = filterByStatus(rows, input.DownloadStatus)
}
if input.Protocol != "" {
    rows = filterByProtocol(rows, input.Protocol)
}
```

`indexer.Service.ListHistory` already takes a limit. Pass a high limit (1000) internally when
filters are active to avoid pagination issues — the result set is always small.

No new SQL queries needed.

### Frontend

In `web/ui/src/pages/history/HistoryPage.tsx`, add a filter bar above the table:
- Status dropdown: All / Completed / Failed / Queued / Downloading
- Protocol dropdown: All / Torrent / NZB
- Sends params to `useHistory(filters)` hook

Extend `useHistory` in `web/ui/src/api/movies.ts` (or `history.ts` if it exists) to accept
optional filter params as query string.

### Tests

No new backend tests needed (filter logic is trivial Go); existing integration tests cover
the endpoint shape.

---

## 3. Bulk Movie Editor

### What it does
On the movie list (Dashboard or a Movies page), user can:
1. Check a checkbox on movie cards to select them
2. A floating toolbar appears: "X selected — Edit"
3. A small modal lets them change: Monitored (on/off), Quality Profile, Minimum Availability
4. "Apply" calls `PUT /movies/{id}` for each selected movie

### Backend

No new endpoints. Uses existing `PUT /api/v1/movies/{id}`.

### Frontend

The main movie list is the Dashboard (`src/pages/Dashboard.tsx`) or a dedicated movies page.
Check where the movie cards are rendered.

Changes:
- Add `selectionMode` state (bool) and `selectedIds` state (Set<string>) to the movie list
- Each movie card gets a checkbox (visible when selectionMode is on, or on hover)
- "Select" toggle button in the page header enables selection mode
- When any are selected: show a sticky bottom bar with count + "Edit Selected" button
- Edit modal has three optional fields (leave blank = don't change):
  - Monitored: toggle (tristate: unchanged / on / off)
  - Quality Profile: dropdown (options from `useQualityProfiles()`) + "unchanged" default
  - Min. Availability: dropdown + "unchanged" default
- Apply: iterate `selectedIds`, call `updateMovie.mutateAsync(...)` for each
- Show progress (e.g. "Updating 5/12…") and success/error toast

**No new API hooks needed** — uses existing `useUpdateMovie`.

---

## 4. Backup / Restore

### What it does

**Backup:** `GET /api/v1/system/backup` streams the SQLite database as a file download.
Uses SQLite's online backup API (via `database/sql` + crawshaw/sqlite or `VACUUM INTO`)
so the backup is consistent even under concurrent writes.

**Restore:** `POST /api/v1/system/restore` accepts the backup file (multipart or raw body),
writes it to a staging path, responds with 200 + a JSON message instructing the user to
restart. On next startup, the main function checks for the staging file and swaps it in.

### Backend — Backup

In `internal/api/v1/system.go` (alongside existing system routes):

```go
// GET /api/v1/system/backup
// Streams the SQLite DB as a file download.
```

Use `VACUUM INTO '/tmp/luminarr-backup-<timestamp>.db'` to create a consistent copy,
then `http.ServeContent` / `io.Copy` to stream it, then `os.Remove` to clean up.

`VACUUM INTO` works on SQLite 3.27+ (available in `modernc.org/sqlite` used here).

Response headers:
```
Content-Type: application/octet-stream
Content-Disposition: attachment; filename="luminarr-backup-2026-03-04.db"
```

**Problem:** huma wraps everything in JSON. For streaming a binary file we need to
bypass huma and write directly to the `http.ResponseWriter`. Register the route
with the underlying chi router, not `huma.Register`.

In `internal/api/router.go`, after building the huma API, add:
```go
r.Get("/api/v1/system/backup", backupHandler(cfg.DB, cfg.Logger))
```

where `cfg.DB` is the `*sql.DB` instance.

### Backend — Restore

```go
// POST /api/v1/system/restore
// Accepts application/octet-stream body; writes to staging path; responds with restart instructions.
```

Also on chi directly (binary body, not JSON).

Staging path: same directory as the DB, named `luminarr.db.restore`.

In `cmd/luminarr/main.go`, at startup before `goose.Up`:
```go
if _, err := os.Stat(dbPath + ".restore"); err == nil {
    if err := os.Rename(dbPath+".restore", dbPath); err != nil {
        log.Fatal("restore swap failed:", err)
    }
    log.Println("database restored from backup")
}
```

**Config struct needs:** `cfg.DBPath string` exposed so the handler knows where to write
the staging file. Currently DBPath is computed in `main.go` — expose it via the config struct
or pass it as a handler parameter.

### Frontend

In `SystemPage.tsx` (or a new "Backup" section in Settings):
- "Download Backup" button — calls `GET /system/backup`, browser downloads the file
- "Restore from Backup" — file input (`.db` extension), `POST /system/restore`, shows
  "Restart Luminarr to complete the restore" message after success

For download: `window.location.href = '/api/v1/system/backup?key=...'` — easiest approach
since huma's `apiFetch` isn't suited for binary blobs. Alternatively fetch + blob URL.

For upload: `FormData` with the file, `fetch` with raw body.

### Tests

**Integration test** (`internal/api/v1/system_backup_test.go`):
- `TestBackup` — GET returns 200 with Content-Disposition header and non-empty body
- `TestRestore` — POST with valid DB bytes returns 200 with JSON message;
  staging file exists at expected path

---

## 5. Interactive Import UI

### What it does

A page (or modal) where users can see video files that are on disk but not yet imported,
match each to a movie, and import them in one click.

This uses two existing endpoints:
- `GET /api/v1/libraries/{id}/disk-scan` — returns untracked files (path, size, matched_movie_id)
- `POST /api/v1/libraries/{id}/import-file` — body: `{ file_path, tmdb_id }`

The `import-file` handler does everything: looks up TMDB, creates the movie if it doesn't
exist, links the file. If the movie already exists (by TMDB ID), it just links the file.

### Frontend

`src/pages/settings/libraries/InteractiveImportPage.tsx` — or a modal accessible from
LibraryList.tsx via a "Import Files" button per library.

**Preferred: modal accessible from LibraryList, not a full page.** Keeps navigation simple.

Flow:
1. User clicks "Import Files" button on a library card in LibraryList
2. Modal opens, shows loading skeleton
3. Fetches `GET /libraries/{id}/disk-scan`
4. For each untracked file, shows:
   - Filename (truncated)
   - Size
   - Parsed title + year (from filename — can compute client-side using same heuristics,
     or add a lightweight backend call to `/movies/suggestions` with filename)
   - "Match" dropdown/search (calls `POST /movies/lookup` to search TMDB)
   - Once matched: show movie title + year + poster thumbnail
   - "Import" button (calls `POST /libraries/{id}/import-file`)
5. Each row has independent state (matching, importing, done, error)
6. "Import All Matched" button at top to batch all rows that have a TMDB ID selected

**Simplification for Phase 19:** The matching UX reuses `useLookupMovies` (already written)
and the same result picker from `MatchTMDBBanner`. No new API needed.

**Note on parsed title:** To avoid reimplementing filename parsing in JS, add a lightweight
endpoint:

```
GET /api/v1/parse?filename=THE_HUNGERGAMES_MOCKINGJAY_PT1.mkv
```

Returns `{ title: "The Hungergames Mockingjay Part 1", year: 0 }`. Calls `ParseFilename`.

Register in a new `internal/api/v1/parse.go` (3 lines of code).

### Tests

**No new backend tests** (disk-scan and import-file already tested; parse endpoint is trivial).
Frontend: manual smoke test.

---

## Key Files Modified

| File | Change |
|---|---|
| `internal/api/v1/queue.go` | Add `POST /{id}/blocklist` endpoint |
| `internal/api/router.go` | Pass blocklist service to `RegisterQueueRoutes` |
| `internal/db/queries/sqlite/blocklist.sql` | Add `IsBlocklistedByTitle` query |
| `internal/api/v1/history.go` | Add `download_status` + `protocol` filter params |
| `internal/api/router.go` | Register chi-level backup/restore handlers |
| `internal/api/v1/parse.go` | New file: `GET /parse` endpoint |
| `cmd/luminarr/main.go` | Startup restore-staging check |
| `web/ui/src/api/queue.ts` | `useBlocklistQueueItem` hook |
| `web/ui/src/pages/queue/Queue.tsx` | Blocklist button per row |
| `web/ui/src/pages/history/HistoryPage.tsx` | Filter bar |
| `web/ui/src/pages/Dashboard.tsx` | Bulk select + edit toolbar |
| `web/ui/src/pages/settings/system/SystemPage.tsx` | Backup/Restore section |
| `web/ui/src/pages/settings/libraries/LibraryList.tsx` | "Import Files" button + modal |

---

## Test Checklist

| Test | File |
|---|---|
| `TestBlocklistQueueItem` | `internal/api/v1/queue_blocklist_test.go` |
| `TestBackup` | `internal/api/v1/system_backup_test.go` |
| `TestRestore` | `internal/api/v1/system_backup_test.go` |
