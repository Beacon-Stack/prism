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

Prism is a self-hosted movie collection manager with a React web UI and a REST API. It tracks a movie library, polls indexers for new releases, grabs them through your download client, and files the finished downloads into your media server. It runs as a single Go binary, stores state in Postgres, and is configured from the UI or through environment variables.

Prism also speaks the Radarr v3 API, so tools like Overseerr, Jellyseerr, Homepage, Home Assistant, and anything else with a "Radarr" integration can connect to it without modification.

Prism is part of the Beacon media stack and runs alongside [Pilot](https://github.com/beacon-stack/pilot) (TV), [Haul](https://github.com/beacon-stack/haul) (BitTorrent), and [Pulse](https://github.com/beacon-stack/pulse) (control plane). Each of those is optional — Prism works on its own too.

## Features

**Library management**

- Full TMDB integration for search, metadata, posters, and collection tracking
- Per-movie monitoring with automatic grab on missing
- Wanted page covering missing movies, cutoff-unmet, and upgrade recommendations
- Calendar view of upcoming releases by month
- Library stats with breakdowns by quality tier, genre, decade, storage trends, and indexer performance — all with clickable drill-down
- Collections browsing and tracking pulled from TMDB

**Release handling**

- Quality profiles with resolution, source, codec, and HDR dimensions
- Custom formats with regex matching and weighted scoring, with TRaSH Guides presets preinstalled
- Edition-aware scoring — 15 canonical editions (Theatrical, Director's Cut, Extended, Ultimate, IMAX, Criterion, 4K Remaster, and others) parsed and scored per movie
- Release decision explainability — every release shows exactly why it was grabbed or skipped, which custom formats matched, and where it landed against the quality profile
- Import conflict detection — warns about dimension regressions (HDR lost, audio downgrade) before grabbing
- Manual search across all indexers with per-release scoring breakdown
- Title-matched release filter — releases for other movies with overlapping words in the title don't get grabbed

**Automation**

- Automatic RSS sync on a configurable schedule
- Auto-search scored against your quality profile and custom formats
- Auto-import of completed downloads with rename and hardlink support
- Import lists from TMDB, Trakt, Plex watchlists, MDBList, and custom URL lists
- Library Sync — compare your media server library against Prism and import the delta
- Activity log pruning

**Integrations**

- **Radarr v3 API compatibility** at `/api/v3/` — usable by Overseerr, Jellyseerr, Homepage, Home Assistant, LunaSea, and any other tool with a "Radarr" integration
- **Indexers:** Newznab (NZBgeek, NZBFinder), Torznab (Prowlarr, Jackett), Pulse-managed indexers
- **Download clients:** [Haul](https://github.com/beacon-stack/haul), qBittorrent, Deluge, Transmission, SABnzbd, NZBGet
- **Media servers:** Plex, Jellyfin, Emby
- **Notifications:** Discord, Slack, Telegram, Pushover, Gotify, ntfy, email, webhook, custom command
- **Migration:** one-click import of quality profiles, libraries, indexers, download clients, and movies from a running Radarr instance

**UI**

- Command palette (Cmd/Ctrl+K) with fuzzy search for pages, movies, and actions
- Interactive release search modal with pack-type filters, quality badges, seed-count column, and per-row scoring
- Dark and light themes with 10+ presets shared across the Beacon services
- Live queue updates over WebSocket
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

The full Beacon stack — Postgres, Pulse, Pilot, Prism, Haul, and a VPN container — is wired up in [`beacon-stack/deploy`](https://github.com/beacon-stack/deploy). Point it at a media directory and go.

### Build from source

Requires Go 1.25+ and Node 22+. The default Docker image includes ffmpeg/ffprobe for media scanning; install it separately if you build locally and want that feature.

```bash
git clone https://github.com/beacon-stack/prism
cd prism
cd web/ui && npm ci && npm run build && cd ../..
make build
./bin/prism
```

Prism listens on port 8282 by default. If something else already owns that port, override with `PRISM_SERVER_PORT`.

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

## Radarr v3 API

Prism exposes a Radarr v3 compatible API at `/api/v3/`. Point any tool that has a "Radarr" integration at Prism:

| Field | Value |
|---|---|
| URL | `http://<prism-host>:8282` |
| API Key | Your Prism API key (Settings → App Settings) |

Currently known to work with Overseerr, Jellyseerr, Homepage, and Home Assistant. If you run into a tool that doesn't work, please file an issue with the tool name and the request path that's failing.

## Migrating from Radarr

Prism imports from a running Radarr instance in one pass. Open Settings → Import, enter the Radarr URL and API key, preview what will be brought over, and pick the categories to bring in. Supported:

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

If the full stack is running, Prism pulls shared indexers and quality profiles from Pulse and sends torrent grabs through Haul. Standalone is fine — Pulse and Haul are optional.

## Power user notes

**TRaSH custom formats.** Prism ships with the commonly-used TRaSH Guides custom format presets preinstalled and scored according to their recommendations. A fresh install gets solid 1080p and 2160p releases without manual configuration. Editing, disabling, and adding formats all work the same as Radarr — the format engine is compatible, and the migration tool brings any existing custom formats along when you import from Radarr.

**Edition-aware scoring.** Prism's parser recognizes 15 canonical editions. Set a preferred edition per movie and matching releases score higher during the decision pass. Useful when you care about a specific cut.

**Release decision explainability.** Every release on the manual search page shows exactly which custom formats matched, which quality tier it landed in, and whether it was blocked by the title match, the quality floor, or a blocklist entry. Makes debugging "why didn't this grab" a two-second operation instead of a mystery.

**Radarr v3 adapter.** Lives in `internal/api/v3/`. It's a compatibility layer over Prism's native API that maps Radarr's endpoint shapes onto Prism's internal types. When a field doesn't have a direct equivalent, the adapter either returns a sensible default or omits it. If you're building or maintaining an external tool that targets Prism through this adapter and something's behaving oddly, file an issue against the specific endpoint.

**API surface.** Everything the UI does is available through the native `/api/v1/` REST API — OpenAPI docs at `/api/docs`.

## Privacy

Prism makes outbound connections only to services you explicitly configure: TMDB for metadata, your indexers, your download clients, your media servers, and your notification targets. No telemetry, no analytics, no crash reporting, no update checks. Credentials are stored locally and never written to logs.

## Built with Claude

Prism was built by one person with extensive help from [Claude](https://claude.ai) (Anthropic). Architecture, design decisions, bug triage, and this README are mine. Many of the keystrokes are not. If something in the code or the docs doesn't make sense, [open an issue](https://github.com/beacon-stack/prism/issues).

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
