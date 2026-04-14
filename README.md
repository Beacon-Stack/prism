<p align="center">
  <h1 align="center">Prism</h1>
  <p align="center">A self-hosted movie collection manager for home servers and the Beacon media stack.</p>
</p>
<p align="center">
  <a href="https://github.com/beacon-stack/prism/blob/main/LICENSE"><img src="https://img.shields.io/github/license/beacon-stack/prism" alt="License"></a>
  <img src="https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white" alt="Go 1.25">
</p>
<p align="center">
  <a href="https://beaconstack.io">Website</a> ·
  <a href="https://github.com/beacon-stack/prism/issues">Bug Reports</a>
</p>

---

Prism is a self-hosted movie collection manager with a React web UI and a REST API. It does what Radarr does — monitors a library, searches indexers, grabs the best available release, imports completed downloads, talks to your media server — but in a single Go binary with a modern UI and sensible defaults. It runs standalone or slots into the Beacon stack alongside [Pilot](https://github.com/beacon-stack/pilot) (TV), [Haul](https://github.com/beacon-stack/haul) (BitTorrent), and [Pulse](https://github.com/beacon-stack/pulse) (control plane).

Prism also speaks the Radarr v3 API, so Overseerr, Jellyseerr, Homepage, Home Assistant, and anything else with a "Radarr" integration can point at Prism without modification.

## Is this for you?

Prism exists for people who love the Radarr ecosystem — Overseerr or Jellyseerr for requests, Homepage or Home Assistant for at-a-glance status, TRaSH Guides for quality tuning — but want a movie manager that's evolving faster than the upstream. Every one of those tools already works with Prism because Prism implements the Radarr v3 API verbatim: point Overseerr's URL at Prism's port, paste in the API key, and it just works. The migration itself is one click — Settings → Import, paste your Radarr URL and API key, and thirty seconds later your quality profiles, libraries, indexers, download clients, and movies are all in Prism with monitoring state intact. Nothing to rebuild, no integrations to relearn.

You'll probably like Prism if you:

- Use Overseerr, Jellyseerr, Homepage, or anything else with a Radarr integration and want to keep using it
- Want TRaSH Guides–grade custom formats preinstalled instead of hand-configuring them
- Care about editions (Theatrical vs Director's Cut vs IMAX) and want Prism to score them rather than ignore them
- Want collection tracking and library analytics that aren't afterthoughts

## Features

**Library management**

- Full TMDB integration for search, metadata, posters, and collection tracking
- Per-movie monitoring with automatic grab on missing
- Wanted page with missing, cutoff-unmet, and upgrade recommendations in one view
- Calendar view of upcoming releases by month
- Library stats with breakdowns by quality tier, genre, decade, storage trends, and indexer performance (with clickable drill-down)
- Collections browsing and tracking from TMDB

**Release handling**

- Quality profiles with resolution, source, codec, and HDR dimensions
- Custom formats with regex matching and weighted scoring — TRaSH Guides presets built in
- Edition support — 15 canonical editions parsed, with per-movie preferred-edition scoring
- Release decision explainability — every release shows exactly why it was grabbed or skipped
- Import conflict detection — warns about dimension regressions (HDR lost, audio downgrade) before grabbing
- Manual search across all indexers with full scoring breakdown
- Strict title matching in the release filter, so "The Dark Knight" won't accidentally grab "The Dark Knight Rises"

**Automation**

- Automatic RSS sync on a configurable schedule
- Auto-search that scores every candidate against your profile and custom formats before grabbing
- Auto-import of completed downloads with rename and hardlink support
- Import lists from TMDB, Trakt, Plex watchlists, MDBList, and custom URL lists
- Library Sync — compare your media server library against Prism and import the delta
- Activity log pruning

**Integrations**

- **Radarr v3 API compatibility** — point Overseerr, Jellyseerr, Homepage, Home Assistant, LunaSea, and any other Radarr-aware tool directly at Prism with no changes
- **Indexers:** Newznab (NZBgeek, NZBFinder), Torznab (Prowlarr, Jackett), Pulse-managed indexers
- **Download clients:** [Haul](https://github.com/beacon-stack/haul), qBittorrent, Deluge, Transmission, SABnzbd, NZBGet
- **Media servers:** Plex, Jellyfin, Emby
- **Notifications:** Discord, Slack, Telegram, Pushover, Gotify, ntfy, email, webhook, custom command

**UI**

- Command palette (Cmd/Ctrl+K) with fuzzy search for pages, movies, and actions
- Interactive release search modal with pack-type filters, quality badges, seed-count column, and per-row scoring
- Dark and light themes with 10+ presets shared across the Beacon services
- Live queue updates over WebSocket — no polling
- OpenAPI documentation at `/api/docs`

**Operations**

- Single static Go binary, no runtime dependencies
- Postgres backend
- Zero telemetry, no analytics, no crash reporting, no phoning home
- Auto-generated API key on first run
- Graceful shutdown with drain timeout

## Getting started

### Docker

```bash
docker run -d \
  --name prism \
  -p 8282:8282 \
  -v /path/to/config:/config \
  -v /path/to/movies:/movies \
  ghcr.io/beacon-stack/prism:latest
```

Open `http://localhost:8282`. Prism generates an API key on first run — find it in Settings → App Settings.

### Docker Compose (with the rest of the stack)

The full Beacon stack — Postgres, Pulse, Pilot, Prism, Haul, and a VPN container — is wired up in [`beacon-stack/stack`](https://github.com/beacon-stack/stack). Point it at a media directory and go.

### Build from source

Requires Go 1.25+ and Node 22+. The default Docker image includes ffmpeg/ffprobe for media scanning; install it separately if you build locally and want that feature.

```bash
git clone https://github.com/beacon-stack/prism
cd prism
cd web/ui && npm ci && npm run build && cd ../..
make build
./bin/prism
```

> **Running Radarr too?** Prism uses port 8282, so you can run both side by side during migration.

## Configuration

Most settings live in the web UI. For the ones you'll want at container-start time, use environment variables or a YAML config file at `/config/config.yaml` (also searched at `~/.config/prism/config.yaml` and `./config.yaml`).

| Variable | Default | Description |
|---|---|---|
| `PRISM_SERVER_PORT` | `8282` | Web UI and API port |
| `PRISM_DATABASE_DSN` | — | Postgres DSN (required) |
| `PRISM_AUTH_API_KEY` | auto | API key; autogenerated on first run if unset |
| `PRISM_PULSE_URL` | — | Pulse control-plane URL (optional) |
| `PRISM_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `PRISM_LOG_FORMAT` | `json` | `json` or `text` |

## Radarr API compatibility

Prism exposes a Radarr v3 compatible API at `/api/v3/`. External tools with a "Radarr" integration point directly at Prism with no changes on their side.

| Field | Value |
|---|---|
| **URL** | `http://<prism-host>:8282` |
| **API Key** | Your Prism API key (Settings → App Settings) |

Tested against Overseerr, Jellyseerr, Homepage, and Home Assistant. If you hit something that doesn't work, please file an issue with the tool name and the request path that's failing.

## Radarr migration

Prism imports from a running Radarr instance in one step. Open Settings → Import, enter your Radarr URL and API key, preview what will be brought over, and pick which categories to import. Supported:

- Quality profiles
- Libraries (root folders)
- Indexers
- Download clients
- Movies with monitoring state

## Where Prism fits in the Beacon stack

```
┌─────────────┐     registers      ┌──────────┐
│    Pulse    │◄───────────────────┤  Prism   │
│ (control    │────managed─────────►(movies)  │
│   plane)    │  indexers + profiles│          │
└─────────────┘                    └────┬─────┘
                                        │
                                 grab torrent
                                        ▼
                                   ┌─────────┐
                                   │  Haul   │
                                   │  (BT)   │
                                   └─────────┘
```

Prism runs fine standalone — Pulse and Haul are optional. If you run the full stack, Prism pulls shared indexers and quality profiles from Pulse and sends torrent grabs through Haul.

## Power user notes

**Custom formats with TRaSH presets.** Prism ships with the commonly-used TRaSH Guides custom format presets preinstalled and scored according to their recommendations, so a fresh install already gets solid 1080p and 2160p releases without any manual configuration. Editing, disabling, and adding new formats all work the same as Radarr — the format engine is compatible. If you're coming from a Radarr install with a heavily tuned setup, the migration tool will bring your custom formats with it.

**Edition support.** Prism's parser recognizes 15 canonical editions (Theatrical, Director's Cut, Extended, Ultimate, IMAX, Criterion, 4K Remaster, and others). Set a preferred edition per movie and Prism scores matching releases higher during the decision pass. Useful when you care about a specific cut of a film.

**Release decision explainability.** Every release on the manual search page shows exactly why it was grabbed or skipped — which custom formats matched, which quality tier it landed in, and whether it was blocked by a title mismatch or the quality profile floor. Makes debugging "why didn't it grab this" trivial instead of mysterious.

**Radarr v3 API adapter.** Lives in `internal/api/v3/`. It's a compatibility shim over Prism's native API that maps Radarr's endpoint shapes onto Prism's internal types. When a field doesn't have a direct equivalent, the adapter either returns a sensible default or omits it. If you're building or maintaining an external tool that targets Prism through this adapter, file issues against endpoint shapes that aren't behaving.

**API surface.** Everything the UI does is available through the native `/api/v1/` REST API — OpenAPI docs at `/api/docs`.

## Privacy

Prism makes outbound connections only to services you explicitly configure: TMDB for metadata, your indexers, your download clients, your media servers, and your notification targets. No telemetry, no analytics, no crash reporting, no update checks. Credentials are stored locally and never written to logs.

## Built with Claude

Prism was built by one person with extensive help from [Claude](https://claude.ai) (Anthropic). Architecture, design decisions, bug triage, and this README are mine. Many of the keystrokes are not. If something in the code or the docs doesn't make sense, that's a bug worth reporting — [open an issue](https://github.com/beacon-stack/prism/issues).

## Development

```bash
make build    # compile to bin/prism
make run      # build + run
make dev      # hot reload (requires air)
make test     # go test ./...
make check    # golangci-lint + tsc --noEmit
make sqlc     # regenerate sqlc code
```

## Contributing

Bug reports, feature requests, and pull requests are welcome. Please open an issue before starting anything large.

## License

MIT — see [LICENSE](LICENSE).
