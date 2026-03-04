# Getting Started with Luminarr

This guide walks you through installing Luminarr, configuring it, and getting your first movies tracked. If you're coming from Radarr, there's a one-click import — skip to [Migrating from Radarr](#migrating-from-radarr).

---

## Prerequisites

You need one thing before you start:

- **A TMDB API key** — free at [themoviedb.org/settings/api](https://www.themoviedb.org/settings/api). Sign up, request an API key (choose "Developer"), and you'll have it in two minutes. Without this key, Luminarr runs but can't search for or fetch movie metadata.

---

## Installation

### Docker (recommended)

```bash
docker run -d \
  --name luminarr \
  -p 8282:8282 \
  -v luminarr-data:/config \
  -v /path/to/movies:/movies \
  -e LUMINARR_TMDB_API_KEY=your-tmdb-key \
  ghcr.io/davidfic/luminarr:latest
```

Open `http://localhost:8282`. Done.

On first run, Luminarr generates an API key and saves it to the `/config` volume. It persists across restarts automatically.

### Docker Compose

Create a `docker-compose.yml`:

```yaml
services:
  luminarr:
    image: ghcr.io/davidfic/luminarr:latest
    ports:
      - "8282:8282"
    environment:
      LUMINARR_TMDB_API_KEY: your-tmdb-key
      # Optional: set a fixed API key instead of auto-generating one
      # LUMINARR_AUTH_API_KEY: my-secret-key
    volumes:
      - luminarr-data:/config
      - /path/to/movies:/movies
    restart: unless-stopped

volumes:
  luminarr-data:
```

```bash
docker compose up -d
```

> **Port choice:** Luminarr uses 8282 so it can run alongside Radarr (7878) during migration.

### Build from source

Requires Go 1.25+ and Node.js 20+.

```bash
git clone https://github.com/davidfic/luminarr
cd luminarr
cd web/ui && npm install && npm run build && cd ../..
make build
./bin/luminarr
```

The binary is fully self-contained — it embeds the React frontend. Config defaults to `~/.config/luminarr/config.yaml` and the database to `~/.config/luminarr/luminarr.db`.

---

## Initial Setup

After starting Luminarr, open the UI at `http://localhost:8282` and configure four things:

### 1. Add a library

**Settings → Libraries → Add Library**

A library is a root folder where your movie files live. Each library maps to a directory on disk (e.g. `/movies`). You'll assign a quality profile and optionally set a minimum free space threshold.

### 2. Create a quality profile

**Settings → Quality Profiles → Add Profile**

Quality profiles define what you want. Unlike Radarr's Custom Formats, Luminarr has four explicit dimensions:

| Dimension | Examples |
|-----------|----------|
| Resolution | 720p, 1080p, 2160p |
| Source | WebDL, Bluray, Remux |
| Codec | x264, x265, AV1 |
| HDR | None, HDR10, Dolby Vision |

Pick a preset (e.g. "HD-1080p x265") or build a custom profile. Set a **cutoff** — the quality level where Luminarr stops looking for upgrades.

### 3. Add an indexer

**Settings → Indexers → Add Indexer**

Luminarr supports **Torznab** and **Newznab** protocols. If you use Prowlarr or Jackett, add each indexer with its URL and API key. Click **Test** to verify the connection.

### 4. Add a download client

**Settings → Download Clients → Add Client**

Supported clients:

- **qBittorrent** — host, port, username, password
- **Deluge** — host, port, password

Click **Test** to verify. Luminarr polls the download client for progress and auto-imports completed downloads into your library.

---

## Adding Movies

Once setup is complete:

1. Go to the **Movies** page
2. Click **Add Movie**
3. Search by title — results come from TMDB
4. Pick a quality profile and library
5. Choose whether to start monitoring immediately

Luminarr will search your indexers for available releases during the next RSS sync (every 15 minutes by default), or you can trigger a manual search from the movie detail page.

---

## How the Grab Pipeline Works

1. **RSS sync** (every 15 min) or **manual search** finds releases on your indexers
2. Luminarr scores each release against your quality profile
3. The best matching release is sent to your download client
4. Luminarr polls the download client for progress (visible on the **Queue** page)
5. When the download completes, the **importer** moves or hardlinks the file into your library
6. If notifications are configured, you get alerts at each stage

---

## Migrating from Radarr

If you already run Radarr, you can import everything in one step.

1. Go to **Settings → Import**
2. Enter your Radarr URL (e.g. `http://localhost:7878`) and API key (found in Radarr → Settings → General → Security)
3. Click **Connect & Preview** — Luminarr shows what it found
4. Select categories to import and click **Import**

Luminarr imports in dependency order:
- Quality profiles (mapped to Luminarr's explicit codec/HDR format)
- Libraries (from Radarr root folders)
- Indexers (Torznab and Newznab only)
- Download clients (qBittorrent and Deluge only)
- Movies (duplicates skipped by TMDB ID)

Radarr keeps running during import. Switch over when you're ready.

---

## Notifications (optional)

**Settings → Notifications → Add Notification**

Supported channels: **Discord**, **Webhook**, **Email**. Each can subscribe to specific events:

- Grab started / failed
- Download complete
- Import complete / failed
- Health issue / resolved

---

## Configuration Reference

All settings can live in `config.yaml` or as environment variables (prefixed with `LUMINARR_`, dots become underscores).

| Setting | Default | Env var | Description |
|---------|---------|---------|-------------|
| `server.host` | `0.0.0.0` | `LUMINARR_SERVER_HOST` | Listen address |
| `server.port` | `8282` | `LUMINARR_SERVER_PORT` | HTTP port |
| `database.driver` | `sqlite` | `LUMINARR_DATABASE_DRIVER` | `sqlite` or `postgres` |
| `database.path` | `~/.config/luminarr/luminarr.db` | `LUMINARR_DATABASE_PATH` | SQLite file path |
| `database.dsn` | — | `LUMINARR_DATABASE_DSN` | Postgres connection string |
| `auth.api_key` | auto-generated | `LUMINARR_AUTH_API_KEY` | API key for all requests |
| `tmdb.api_key` | — | `LUMINARR_TMDB_API_KEY` | TMDB metadata key |
| `log.level` | `info` | `LUMINARR_LOG_LEVEL` | `debug`, `info`, `warn`, `error` |
| `log.format` | `json` | `LUMINARR_LOG_FORMAT` | `json` or `text` |

Config file search order:
1. `/config/config.yaml` (Docker volume mount)
2. `~/.config/luminarr/config.yaml`
3. `/etc/luminarr/config.yaml`
4. `./config.yaml`

A fully commented example is at [`config.example.yaml`](../config.example.yaml).

---

## API Key

Every Luminarr instance has its own randomly generated API key. This is intentional.

**Why one key per instance, not a shared key?**
Tools like Radarr ship with one API key that's visible in the settings UI — it's the same key for every client that talks to that instance, and you copy-paste it wherever needed. That works fine when you're the only user, but it means anyone who has ever seen the key has permanent access.

Luminarr takes the same one-key-per-instance approach, but removes the copy-paste step entirely: the key is generated once on first start and **baked directly into the HTML** that the browser receives. The browser stores it in memory and sends it with every API request automatically. You never see it, never manage it, never paste it anywhere — it just works.

The practical consequence: if you restart the container without a persistent key, the key changes and your open browser tabs get a 401 until you hard-refresh. Fix this with `LUMINARR_AUTH_API_KEY` or by mounting a `config.yaml`. See the [Troubleshooting](#troubleshooting) section.

**Using the API from outside the browser** (scripts, Home Assistant, etc.): find your key in the container logs on startup (`api key: ...`), or set a fixed key via `LUMINARR_AUTH_API_KEY` and use that value in the `X-Api-Key` header.

Interactive OpenAPI docs are available at `/api/docs` when the server is running.

---

## Troubleshooting

### 401 errors after restarting Docker

The API key changed. Either:
- Hard-refresh the browser tab (Ctrl+Shift+R) to pick up the new key
- Set `LUMINARR_AUTH_API_KEY` in your Docker config so the key is stable across restarts

### "TMDB API key not configured" warning

Movie search and metadata are disabled. Set `LUMINARR_TMDB_API_KEY` via environment variable or `tmdb.api_key` in config.yaml.

### Download client connection fails

- Verify the host/port are reachable from the Luminarr container
- In Docker, use the host's IP or Docker network alias — not `localhost`
- Check that your download client's web UI is enabled and credentials are correct

### Indexer test fails

- Check the indexer URL includes the full API path (e.g. `http://prowlarr:9696/1/api`)
- Verify the API key matches what your indexer expects
- Ensure the indexer is reachable from the Luminarr container

---

## Next Steps

- Browse the [Architecture docs](ARCHITECTURE.md) for internals
- Check the [API docs](/api/docs) for automation
- Report bugs or request features on [GitHub](https://github.com/davidfic/luminarr/issues)
