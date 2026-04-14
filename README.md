<p align="center">
  <h1 align="center">Prism</h1>
  <p align="center">A self-hosted movie collection manager built for simplicity.</p>
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

**Prism** monitors your movie library, searches indexers, and automatically grabs the best available release. It is written in Go and React, starts in under a second, and idles under 60 MB of RAM.

Prism is part of the [Beacon](https://beaconstack.io) media stack — it runs alongside [Pilot](https://github.com/beacon-stack/pilot) (TV), [Haul](https://github.com/beacon-stack/haul) (BitTorrent downloader), and [Pulse](https://github.com/beacon-stack/pulse) (control plane) — but it also runs standalone if you only want a movie manager.

If you are coming from Radarr, Prism can import your quality profiles, libraries, indexers, download clients, and movie list in one step.

## Features

**Library management**

- Full TMDB integration for search, metadata, posters, and collection tracking
- Per-movie monitoring with automatic grab on missing
- Wanted page with missing, cutoff-unmet, and upgrade recommendations in one view
- Calendar view of upcoming releases by month
- Library statistics with breakdown by quality tier, genre, decade, storage trends, and indexer performance
- Collections browsing and tracking from TMDB

**Quality and release handling**

- Quality profiles with resolution, source, codec, and HDR dimensions
- Custom formats with regex matching and weighted scoring, TRaSH Guides presets built in
- Edition support — 15 canonical editions parsed, with per-movie preferred-edition scoring
- Release decision explainability — "why was this grabbed or skipped?" surfaced on every row
- Import conflict detection — warns about dimension regressions (HDR lost, audio downgrade) before grabbing
- Manual search across all indexers with full scoring breakdown

**Automation**

- Automatic RSS sync on a configurable schedule
- Auto-search that scores every candidate against your profile and custom formats before grabbing
- Auto-import of completed downloads with rename + hardlink support
- Import lists from TMDB, Trakt, Plex watchlists, MDBList, and custom URL lists
- Library sync — compare your media server against Prism and import the delta
- Activity log pruning

**Integrations**

Indexers:
- Newznab (NZBgeek, NZBFinder, etc.)
- Torznab (Prowlarr, Jackett)
- [Pulse](https://github.com/beacon-stack/pulse) — centrally managed indexers pushed from the Pulse control plane

Download clients:
- [Haul](https://github.com/beacon-stack/haul) — first-class integration with the Beacon torrent client
- qBittorrent, Deluge, Transmission
- SABnzbd, NZBGet

Media servers:
- Plex, Jellyfin, Emby

Notifications:
- Discord, Slack, Telegram, Pushover, Gotify, ntfy
- Email (SMTP with STARTTLS/TLS)
- Webhook (generic HTTP)
- Custom command/script execution

**Compatibility**

- **Radarr v3 API** — Prism exposes a Radarr v3-compatible API at `/api/v3/`, so Overseerr, Jellyseerr, Homepage, Home Assistant, LunaSea and other tools that speak Radarr can talk to Prism without changes.

**UI**

- Command palette (Cmd/Ctrl+K) with fuzzy search for pages, movies, and actions
- Interactive release search modal with pack-type filters, quality badges, and per-row scoring
- Theme system with dark and light modes, 10+ presets shared across the Beacon services
- Directory browser for selecting library root paths
- WebSocket live updates for queue progress
- OpenAPI documentation at `/api/docs`

**Operations**

- Single static binary, no runtime dependencies
- Postgres backend
- Zero telemetry, no analytics, no crash reporting, no phoning home
- Auto-generated API key on first run
- Graceful shutdown with drain timeout

## Getting started

### Docker Compose (recommended, as part of the Beacon stack)

The easiest way to run Prism is as part of the full Beacon stack — see [`beacon-stack/stack`](https://github.com/beacon-stack/stack) for the full docker-compose setup with Postgres, Pulse, Pilot, Prism, and Haul behind a VPN container.

### Standalone Docker

```bash
docker run -d \
  --name prism \
  -p 8282:8282 \
  -v /path/to/config:/config \
  -v /path/to/movies:/movies \
  ghcr.io/beacon-stack/prism:latest
```

Open `http://localhost:8282`. No configuration required to get started.

### Build from source

Requires Go 1.25+ and Node.js 22+.

```bash
git clone https://github.com/beacon-stack/prism
cd prism
cd web/ui && npm ci && npm run build && cd ../..
make build
./bin/prism
```

> **Running Radarr too?** Prism uses port 8282 so you can run both side by side during migration.

## Configuration

Prism works with zero configuration. All settings are editable through the web UI or via environment variables.

### Key environment variables

| Variable | Default | Description |
|---|---|---|
| `PRISM_SERVER_HOST` | `0.0.0.0` | Bind address |
| `PRISM_SERVER_PORT` | `8282` | HTTP port |
| `PRISM_DATABASE_DSN` | | Postgres connection string |
| `PRISM_AUTH_API_KEY` | auto-generated | API key for external access |
| `PRISM_PULSE_URL` | | Pulse control-plane URL (optional) |
| `PRISM_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

### Config file

Prism looks for `config.yaml` in `/config/config.yaml`, `~/.config/prism/config.yaml`, `/etc/prism/config.yaml`, or `./config.yaml` (in that order).

## Radarr API compatibility

Prism exposes a Radarr v3 compatible API at `/api/v3/`. External tools with a "Radarr" integration — Overseerr, Jellyseerr, Homepage, Home Assistant, LunaSea — point directly at Prism with no changes on their side.

| Field | Value |
|---|---|
| **URL** | `http://<prism-host>:8282` |
| **API Key** | Your Prism API key (Settings → App Settings) |

## Radarr migration

Prism can import from a running Radarr instance. Go to Settings → Import, enter your Radarr URL and API key, preview what will be imported, and select which categories to bring over. Supported imports:

- Quality profiles
- Libraries (root folders)
- Indexers
- Download clients
- Movies (with monitoring state)

## Where Prism fits in the Beacon stack

```
┌─────────────┐     registers      ┌──────────┐
│    Pulse    │◄───────────────────┤  Prism   │
│ (control    │────managed─────────►(Movies)  │
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

Prism is fine standalone — Pulse and Haul are optional dependencies. If you run the full stack, Prism pulls its indexers and quality profiles from Pulse, and sends torrent grabs through Haul.

## Privacy

Prism makes outbound connections only to services you explicitly configure: TMDB (for metadata), your indexers, your download clients, your media servers, your notification targets, and optionally the Anthropic API for AI features. No telemetry, no analytics, no crash reporting, no update checks. Credentials are stored locally and never written to logs.

## Project structure

```
cmd/prism/          Entry point
internal/
  api/              HTTP router, middleware, v1 handlers, v3 Radarr adapter
  config/           Configuration loading
  core/             Domain services (movie, quality, library, queue, etc.)
  db/               Database migrations and generated query code (sqlc)
  parser/           Release title parser (quality, edition, language)
  pulse/            Pulse control-plane integration
  scheduler/        Background job scheduler
plugins/
  downloaders/      Haul, qBittorrent, Deluge, Transmission, SABnzbd, NZBGet
  importlists/      TMDB, Trakt, Plex watchlist, MDBList, custom list
  indexers/         Newznab, Torznab
  mediaservers/     Plex, Jellyfin, Emby
  notifications/    Discord, Slack, Telegram, Pushover, Gotify, ntfy, email, webhook, command
web/ui/             React 19 + TypeScript + Vite frontend
```

## Development

```bash
make build         # compile binary to bin/prism
make run           # build + run
make dev           # hot reload with air
make test          # go test ./...
make check         # golangci-lint + tsc --noEmit
make sqlc          # regenerate SQLC code
```

## Contributing

Bug reports, feature requests, and pull requests are welcome. Please open an issue before starting large changes.

## License

MIT
