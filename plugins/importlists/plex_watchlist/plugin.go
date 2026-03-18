// Package plexwatchlist implements an import list plugin that fetches a user's
// Plex watchlist from the Plex cloud API (metadata.provider.plex.tv).
// NOTE: This requires a Plex *account* token, not a server token.
package plexwatchlist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/safedialer"
	"github.com/luminarr/luminarr/pkg/plugin"
)

const (
	plexBaseURL  = "https://metadata.provider.plex.tv"
	plexClientID = "luminarr"
	plexProduct  = "Luminarr"
	httpTimeout  = 30 * time.Second
)

func init() {
	registry.Default.RegisterImportList("plex_watchlist", func(settings json.RawMessage) (plugin.ImportList, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("plex_watchlist: invalid settings: %w", err)
		}
		if cfg.AccessToken == "" {
			return nil, fmt.Errorf("plex_watchlist: access_token is required (Plex account token, not server token)")
		}
		return &Plugin{cfg: cfg}, nil
	})
	registry.Default.RegisterImportListSanitizer("plex_watchlist", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["access_token"]; ok {
			m["access_token"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the settings for the Plex watchlist plugin.
type Config struct {
	AccessToken string `json:"access_token"`
}

// Plugin fetches movies from a user's Plex watchlist.
type Plugin struct {
	cfg    Config
	client *http.Client

	// tmdbClient is injected by the service for GUID→TMDb ID resolution.
	tmdbLookup func(ctx context.Context, imdbID string) (int, string, error)
}

func (p *Plugin) Name() string { return "Plex Watchlist" }

// SetTMDBClient accepts a *tmdb.Client for resolving Plex GUIDs to TMDb IDs.
func (p *Plugin) SetTMDBClient(c any) {
	type findByIMDb interface {
		FindByIMDbID(ctx context.Context, imdbID string) (int, string, error)
	}
	if tc, ok := c.(findByIMDb); ok {
		p.tmdbLookup = tc.FindByIMDbID
	}
}

func (p *Plugin) httpClient() *http.Client {
	if p.client != nil {
		return p.client
	}
	return &http.Client{
		Timeout:   httpTimeout,
		Transport: safedialer.Transport(), // cloud endpoint, not LAN
	}
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	items, err := p.fetchWatchlist(ctx)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	_, err := p.fetchWatchlist(ctx)
	return err
}

// watchlistResponse is the Plex metadata API response for watchlist items.
type watchlistResponse struct {
	MediaContainer struct {
		Metadata []plexItem `json:"Metadata"`
	} `json:"MediaContainer"`
}

type plexItem struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	Type  string `json:"type"`
	GUID  string `json:"guid"` // e.g. "plex://movie/..."
	Guids []struct {
		ID string `json:"id"` // e.g. "imdb://tt1234567", "tmdb://12345"
	} `json:"Guid"`
}

func (p *Plugin) fetchWatchlist(ctx context.Context) ([]plugin.ImportListItem, error) {
	reqURL := plexBaseURL + "/library/sections/watchlist/all?type=1&includeGuids=1"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", p.cfg.AccessToken)
	req.Header.Set("X-Plex-Client-Identifier", plexClientID)
	req.Header.Set("X-Plex-Product", plexProduct)

	resp, err := p.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plex API returned http %d", resp.StatusCode)
	}

	var body watchlistResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	items := make([]plugin.ImportListItem, 0, len(body.MediaContainer.Metadata))
	for _, m := range body.MediaContainer.Metadata {
		if m.Type != "" && m.Type != "movie" {
			continue
		}
		item := plugin.ImportListItem{
			Title: m.Title,
			Year:  m.Year,
		}

		// Extract TMDb/IMDb IDs from Guid array.
		for _, g := range m.Guids {
			if len(g.ID) > 7 && g.ID[:7] == "tmdb://" {
				if id, err := strconv.Atoi(g.ID[7:]); err == nil {
					item.TMDbID = id
				}
			}
			if len(g.ID) > 7 && g.ID[:7] == "imdb://" {
				item.IMDbID = g.ID[7:]
			}
		}

		// If no TMDb ID but we have IMDb, try to resolve via TMDb find.
		if item.TMDbID == 0 && item.IMDbID != "" && p.tmdbLookup != nil {
			if tmdbID, _, err := p.tmdbLookup(ctx, item.IMDbID); err == nil && tmdbID > 0 {
				item.TMDbID = tmdbID
			}
		}

		if item.TMDbID == 0 {
			continue // Can't add without a TMDb ID
		}

		items = append(items, item)
	}
	return items, nil
}
