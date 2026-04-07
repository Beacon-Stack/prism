package mdblist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/beacon-stack/prism/internal/registry"
	"github.com/beacon-stack/prism/internal/safedialer"
	"github.com/beacon-stack/prism/pkg/plugin"
)

func init() {
	registry.Default.RegisterImportList("mdblist", func(settings json.RawMessage) (plugin.ImportList, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("mdblist: invalid settings: %w", err)
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("mdblist: api_key is required")
		}
		if cfg.ListID == "" {
			return nil, fmt.Errorf("mdblist: list_id is required")
		}
		return &Plugin{
			cfg: cfg,
			http: &http.Client{
				Transport: safedialer.Transport(),
				Timeout:   30 * time.Second,
			},
		}, nil
	})
	registry.Default.RegisterImportListSanitizer("mdblist", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["api_key"]; ok {
			m["api_key"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the settings for the MDBList plugin.
type Config struct {
	APIKey string `json:"api_key"`
	ListID string `json:"list_id"`
}

// Plugin fetches movies from an MDBList list.
type Plugin struct {
	cfg  Config
	http *http.Client
}

func (p *Plugin) Name() string { return "MDBList" }

// listItem is a single item from the MDBList API response.
type listItem struct {
	Title  string `json:"title"`
	Year   int    `json:"year"`
	IMDbID string `json:"imdb_id"`
	TMDbID int    `json:"tmdb_id"`
}

func (p *Plugin) Fetch(ctx context.Context) ([]plugin.ImportListItem, error) {
	var all []plugin.ImportListItem
	offset := 0
	const limit = 1000

	for {
		url := fmt.Sprintf("https://api.mdblist.com/lists/%s/items?apikey=%s&limit=%d&offset=%d",
			p.cfg.ListID, p.cfg.APIKey, limit, offset)

		items, err := p.fetchPage(ctx, url)
		if err != nil {
			return nil, err
		}

		for _, it := range items {
			if it.TMDbID == 0 {
				continue
			}
			all = append(all, plugin.ImportListItem{
				TMDbID: it.TMDbID,
				IMDbID: it.IMDbID,
				Title:  it.Title,
				Year:   it.Year,
			})
		}

		if len(items) < limit {
			break
		}
		offset += limit
	}
	return all, nil
}

func (p *Plugin) Test(ctx context.Context) error {
	url := fmt.Sprintf("https://api.mdblist.com/lists/%s/items?apikey=%s&limit=1",
		p.cfg.ListID, p.cfg.APIKey)
	_, err := p.fetchPage(ctx, url)
	return err
}

func (p *Plugin) fetchPage(ctx context.Context, url string) ([]listItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("mdblist: %w", err)
	}

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mdblist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("mdblist: rate limited (429)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("mdblist: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var items []listItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("mdblist: decode: %w", err)
	}
	return items, nil
}
