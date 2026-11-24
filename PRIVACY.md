# Privacy

Prism collects nothing. There is no telemetry, no analytics, no crash
reporting and no account. Everything it knows stays in its database and
config file on your machine.

## What Prism connects to

| Destination | When |
|---|---|
| **TMDB** | Searching for movies and fetching their details and artwork |
| **Your indexers** | RSS sync and release searches |
| **Your download clients** | Sending grabs and reading the queue |
| **Your media servers** (Plex, Jellyfin, Emby) | Library refresh and watch status, if you add one |
| **Your import lists** (Trakt, TMDB lists, Plex Watchlist and others) | Syncing a list, if you add one |
| **Your notification channels, or Pulse** | Sending notifications, if you set them up |
| **Pulse** | Shared indexers, download clients and quality profiles, if you connect Prism to it |
| **GitHub** | Only when you press **Check for updates** on the System page |

Your browser also loads the interface fonts from Google Fonts. If it
can't reach them, the interface falls back to a system font.

## Credentials

- API keys and passwords are stored in your local `config.yaml` and
  database, and sent only to the service they belong to.
- They are kept out of logs: secrets render as `***`, and logged URLs
  have their keys removed.

## Other services

A service you connect (TMDB, an indexer, a download client) has its own
privacy policy. This page covers only what Prism itself does. It
describes the software and is not a legal agreement; the source code is
the reference.
