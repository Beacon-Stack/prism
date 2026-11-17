# Plan 36 — Stall Handling + Manual-Grab Freedom

## Problem

Two live incidents (2026-08-30) exposed the same design hole from two angles:

1. **Outlander (2008)** — user grabbed a release, it stalled at 0%, Haul
   paused it (hiding it from Haul's default torrent view). Prism's
   grab_history row sat in `paused` forever, and the
   `idx_grab_history_active_movie` unique partial index blocked every
   subsequent grab for the movie with:
   *"already downloading another release for this movie … Remove it from
   the queue first."*
2. **No Hard Feelings (2023)** — identical state since **2026-05-07**.
   Paused at 0% in Haul, `paused` in grab_history, movie silently
   ungrabbable for almost four months. Nobody noticed because nothing
   surfaces it.

Root causes, in order of importance:

- **No non-terminal state ever times out.** `queued`, `downloading`,
  `paused` grabs live forever. The only reconciliation that exists is
  `queue.Service.pollClient`'s `ErrItemNotFound` branch (item deleted
  from the client → mark `removed`). A torrent that still *exists* in
  the client but will never finish — stalled, paused-at-0%, dead
  swarm — is invisible. Pilot solved this with `stallwatcher`
  (`pilot/internal/core/stallwatcher/`); Prism never got the port.
- **The one-active-grab guard applies to manual grabs.** The index was
  designed (plan 35, "Race Condition Mitigation — Active Grab Dedup")
  to stop RSS sync and auto-search racing each other into duplicate
  downloads. Applying it to deliberate user grabs was an accident of
  the schema — a stuck download becomes a lock on the whole movie.
- **The 409 is a dead end.** It names the blocking release but offers
  no action; the user has to find the item in Prism's queue themselves
  (and Haul's UI actively hides paused torrents, sending them to the
  wrong place first).

## Design

Three phases, each independently shippable, in this order.

---

## Phase 1 — Stallwatcher port (root cause)

Port Pilot's stallwatcher to `prism/internal/core/stallwatcher/`,
adapted movie-shaped. Pilot's `service.go` + `service_test.go` are the
template; keep the same design decisions:

- **Pull-based**: poll Haul `GET /api/v1/stalls` every 60s. The
  endpoint shape is an inter-service contract (see `haul/CLAUDE.md`);
  Prism's copy of `StalledTorrent` must match
  `pilot/plugins/downloaders/haul/haul.go` exactly.
- **Correlation key**: Haul keys stalls by info_hash. For Haul grabs,
  `grab_history.client_item_id` *is* the info hash (verified against
  the live incidents: `897c4d28…`, `2cca50a3…`). No schema change
  needed for correlation. Non-Haul clients (qBittorrent) are out of
  scope for the stall poll but covered by the stale sweep below.
- **On stall**: blocklist the release (`blocklistSvc.Add` with reason
  `stall_<type>`, e.g. `stall_no_peers_ever`), set
  `download_status = 'stalled'`, fire an activity/notification event.
- **`stalled` is terminal.** Add it nowhere to the NOT-IN lists —
  i.e. it stays outside `('completed','failed','removed')` today, so
  every "active" filter must be updated (see Migration below).
- **Auto-retry with circuit breaker**: if the grab came from
  auto-search/RSS (needs `grab_origin` from Phase 2; until then, retry
  nothing), trigger a re-search for the movie. Cap: 3 stall blocklists
  per movie per 24h (`CountRecentStalls` scoped by movie_id — Pilot
  scopes by (series, episode); movies are simpler).
- **Startup grace**: no blocklisting in the watcher's first 2 minutes
  (absorbs Prism-restart races, same as Pilot).
- **Stale sweep** (covers what Haul's stall endpoint can't see):
  - grab in `queued`/`downloading`/`pending` with no progress for
    `StaleGrabAge` (24h) → mark `failed`, blocklist, log.
  - grab in `paused` with `downloaded_bytes == 0` for
    `PausedStallAge` (24h) → treat as a stall: blocklist + `stalled`.
    A 0%-paused torrent is stall evidence, not a user preference —
    both incidents were exactly this shape. A paused grab **with**
    progress is left alone indefinitely (genuine user pause).

~~Open question~~ **Resolved (2026-08-30, verified against the live
deployment):** Haul *auto-pauses* stalled torrents (status flips to
`paused` with `stalled_at` set — that's what happened to both incident
torrents), and once paused they **drop out of `/api/v1/stalls`** (the
endpoint returned `[]` while both sat paused-at-0% with `stalled_at`
set). Consequence: the stalls poll only catches a stall in the window
between detection and auto-pause, so **the paused-at-0% sweep is the
primary detection path**, not the fallback. Implement the sweep first;
the `/stalls` poll is the fast path for stalls Haul hasn't paused yet.
(Related fix, shipped separately in Haul: stalled torrents no longer
vanish from the UI's default Active view —
`haul` branch `fix/stalled-torrents-visible`.)

### Staleness latency: handle each failure mode where it hurts

Third live incident (2026-08-31, A House of Dynamite): user deleted a
dead torrent in Haul, immediately grabbed a replacement, and hit the
409 — the second grab landed inside queue_poll's 60s window before the
`ErrItemNotFound` reconciliation marked the old grab `removed`. The fix
is NOT a generic push-event system; each failure mode has a different
right answer, sorted by *when staleness actually hurts*:

- **Deleted in client** — only hurts at grab time, so resolve it
  synchronously at grab time: when the active-grab precheck in
  `internal/api/v1/releases.go` finds a blocking grab, call
  `client.Status()` live before refusing; on
  `plugin.ErrItemNotFound`, mark the grab `removed` and let the new
  grab proceed. Zero-latency exactly when it matters, no event infra,
  immune to Prism downtime. (After Phase 2 the manual path skips the
  guard entirely, so this reconcile only ever fires for the auto
  path — keep it anyway.)
- **Stalled** — no user action ever asks the right question, so it
  must be *noticed*: the paused-at-0% sweep (reliable path) plus a
  fast path — subscribe to Haul's existing WebSocket event feed
  (Haul already publishes `TypeTorrentStalled` /
  `TypeTorrentRemoved` to its WS hub for its own UI; consumers hold
  Haul's URL + API key already, so the dependency direction is
  preserved). Events are a "run the stallwatcher tick now" signal
  only — never trusted as truth; the tick re-derives state from
  Haul's API. Lost event ⇒ the sweep catches it; duplicate ⇒ a
  harmless extra tick.
- **Completed** — 60s polling stays; nothing is blocked on import
  latency.

### Migration `00002_stalled_status_grab_origin.sql`

`stalled` must free the movie, so every active-grab filter changes in
lockstep (SQLite: drop + recreate the partial index):

```sql
DROP INDEX idx_grab_history_active_movie;
CREATE UNIQUE INDEX idx_grab_history_active_movie
ON grab_history (movie_id)
WHERE download_status NOT IN ('completed','failed','removed','stalled')
  AND grab_origin = 'auto';          -- Phase 2, same migration

ALTER TABLE grab_history ADD COLUMN grab_origin TEXT NOT NULL DEFAULT 'manual';
```

(Column added before the index that references it; shown out of order
for readability. Backfill: historical rows default `'manual'` — the
index only needs to hold for *future* automation races.)

Queries to update in `internal/db/queries/sqlite/indexers.sql` (and the
postgres variants when that lands):

- `ListActiveGrabs` — add `'stalled'` to the NOT-IN list.
- `ActiveGrabForMovie` — same, **and** it must become `:many`
  (Phase 2 allows several concurrent manual grabs).
- Any queue/status query filtering on the terminal set — grep for
  `NOT IN ('completed'` across `internal/db/queries/`.
- New: `CountRecentStallsForMovie`, `MarkGrabStalled`.

### Tests (regression suite, Pilot pattern)

Port the Pilot suite shape — mock querier + httptest fake Haul, no
build tags, runs in `go test ./...`:

- `TestTick_BlocklistsStalledGrab` — headline end-to-end.
- `TestTick_SkipsNonMatchingInfoHash`
- `TestTick_RespectsStartupGrace`
- `TestTick_SkipsAlreadyTerminalGrabs` (idempotency)
- `TestTick_CircuitBreaker` / `TestTick_AutoSearchTriggersRetry`
- `TestTick_ManualDoesNotAutoRetry` (Prism analogue of
  Pilot's interactive-grab gating)
- `TestTick_NoHaulConfigured` / `TestTick_HaulUnreachable`
- `TestSweep_PausedAtZeroIsStall` / `TestSweep_PausedWithProgressLeftAlone`
  / `TestSweep_StaleQueuedGrabFails` — new, Prism-specific.

Add the stallwatcher files to Prism's `CLAUDE.md` regression-guard
list, mirroring Pilot's.

---

## Phase 2 — `grab_origin`: manual grabs bypass the guard

- Every grab path stamps origin: `releases.go` grab handler →
  `'manual'`; `autosearch` + `rss_sync` → `'auto'`.
- The unique index is scoped `WHERE grab_origin = 'auto'` (see
  migration above). Automation keeps its TOCTOU protection — the
  entire original point of plan 35's index — but a user's deliberate
  grab never conflicts with anything.
- Delete the 409 precheck for manual grabs in
  `internal/api/v1/releases.go` (~line 346). Keep it for nothing else —
  auto paths already handle the conflict as `"skipped: already
  downloading"` via the unique-violation branch in `indexerSvc.Grab`.
- `ActiveGrabForMovie` callers audit: queue status, movie "downloading"
  pill, `movieEligibleForGrab` in rss_sync — all must tolerate >1
  active grab (rss_sync: *any* active grab still means "don't
  auto-grab", origin irrelevant).

### Import semantics with concurrent grabs

Two (or more) downloads for one movie can now both complete. Defined
behavior: **every completed download runs through the existing upgrade
logic** in the importer:

- No file on record → import normally.
- File exists → compare quality via the movie's profile
  (`WantRelease` / score): better → replace (current upgrade path);
  equal or worse → skip the import, mark the grab `completed`, log
  `import skipped: not an upgrade`, and leave the download to the
  seed-lifecycle cleanup (seedenforcer). No new machinery — this is
  the same decision the RSS upgrade path already makes, applied at
  import time.

Test: `TestImport_SecondCompletedGrabIsUpgradeDecision` — two grabs,
lower-quality one completes second, file on disk unchanged.

---

## Phase 3 — Escape-hatch UX

- The (now auto-only) 409 and the queue API expose the blocking grab's
  id + title so the UI can render the conflict usefully.
- Queue page: `stalled` rows get a distinct badge + "why" tooltip
  (stall reason from the blocklist entry).
- Manual search modal: if an active grab exists for the movie, show an
  informational banner ("already downloading <title> — grabbing another
  release will race it; best quality wins at import") instead of any
  hard block. Optional per-row action: "replace existing download"
  (queue-remove + grab, one click) for users who *don't* want to race.
- Activity/notification event when the stallwatcher acts, so a stalled
  grab is never silent again (the 4-month No Hard Feelings limbo must
  be impossible to repeat).

---

## Rollout / remediation

1. Phase 1 + migration ship together (index recreate requires the
   `stalled` status to exist in code first). Phase 2 rides the same
   migration but the handler/UI changes can land separately.
2. On first deploy, the stallwatcher's paused-at-0% sweep will
   automatically clean up the two incident grabs (Outlander,
   No Hard Feelings) — verify both movies become grabbable and both
   releases land on the blocklist.
3. Coordinate nothing with Haul unless the Phase 1 open question
   (auto-pause vs. `/api/v1/stalls` visibility) says otherwise; the
   stalls endpoint contract is already consumed by Pilot and must not
   change shape unilaterally.

## Non-goals

- Multi-version libraries ("keep 1080p and 4K") — Prism still tracks
  one file per movie; the loser of a race gets cleaned up, not kept.
- Outbound webhooks from Haul to consumers — Haul never learns who its
  consumers are. The stall fast path is consumer-initiated (WS
  subscription, see "Staleness latency" above), and state is always
  re-derived by polling, never taken from event payloads.
- Porting any of this back into Pilot (it already has stallwatcher;
  Pilot's episode-scoped one-active-grab guard, if/when it grows one,
  should copy the `grab_origin` scoping from here).
