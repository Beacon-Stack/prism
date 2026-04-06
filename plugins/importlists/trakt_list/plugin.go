package traktlist

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/beacon-media/prism/internal/registry"
	"github.com/beacon-media/prism/internal/trakt"
	"github.com/beacon-media/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("trakt_list", func(settings json.RawMessage) (plugin.ImportList, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("trakt_list: invalid settings: %w", err)
		}
		if cfg.Username == "" {
			return nil, fmt.Errorf("trakt_list: username is required")
		}
		if cfg.ListType == "" {
			cfg.ListType = "watchlist"
		}
		if cfg.ListType == "custom" && cfg.ListSlug == "" {
			return nil, fmt.Errorf("trakt_list: list_slug is required for custom lists")
		}
		return &Plugin{cfg: cfg}, nil
	})
	// No sanitizer needed — no secrets in settings (client comes from injection).
}

// Config holds the settings for the Trakt list plugin.
type Config struct {
	Username string `json:"username"`
	ListType string `json:"list_type"` // "watchlist" or "custom"
	ListSlug string `json:"list_slug"` // required when list_type is "custom"
}

// Plugin fetches movies from a Trakt watchlist or custom list.
// The Trakt client is injected from the app-level Trakt client ID.
type Plugin struct {
	cfg    Config
	client *trakt.Client
}

func (p *Plugin) Name() string { return "Trakt List" }

func (p *Plugin) SetTraktClient(c any) {
	if tc, ok := c.(*trakt.Client); ok {
		p.client = tc
	}
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	if p.client == nil {
		return nil, fmt.Errorf("Trakt client not configured — set trakt.client_id in config")
	}

	var movies []trakt.Movie
	var err error

	switch p.cfg.ListType {
	case "custom":
		movies, err = p.client.GetCustomList(ctx, p.cfg.Username, p.cfg.ListSlug)
	default: // "watchlist"
		movies, err = p.client.GetWatchlist(ctx, p.cfg.Username)
	}
	if err != nil {
		return nil, err
	}

	items := make([]plugin.ImportListItem, 0, len(movies))
	for _, m := range movies {
		if m.IDs.TMDB == 0 {
			continue
		}
		items = append(items, plugin.ImportListItem{
			TMDbID: m.IDs.TMDB,
			IMDbID: m.IDs.IMDB,
			Title:  m.Title,
			Year:   m.Year,
		})
	}
	return items, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("Trakt client not configured — set trakt.client_id in config")
	}
	switch p.cfg.ListType {
	case "custom":
		_, err := p.client.GetCustomList(ctx, p.cfg.Username, p.cfg.ListSlug)
		return err
	default:
		_, err := p.client.GetWatchlist(ctx, p.cfg.Username)
		return err
	}
}
