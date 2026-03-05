// Package jellyfin implements a Luminarr media server plugin for Jellyfin.
// On import_complete it triggers a full library refresh.
package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/internal/safedialer"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterMediaServer("jellyfin", func(settings json.RawMessage) (plugin.MediaServer, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("jellyfin: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf("jellyfin: url is required")
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("jellyfin: api_key is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterMediaServerSanitizer("jellyfin", func(settings json.RawMessage) json.RawMessage {
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

// Config holds the user-supplied settings for a Jellyfin server.
type Config struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// Server is a Jellyfin media server plugin instance.
type Server struct {
	cfg    Config
	client *http.Client
}

// New creates a new Server from the given config.
func New(cfg Config) *Server {
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	return &Server{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second, Transport: safedialer.LANTransport()},
	}
}

func (s *Server) Name() string { return "Jellyfin" }

// RefreshLibrary triggers a full library refresh on the Jellyfin server.
func (s *Server) RefreshLibrary(ctx context.Context, _ string) error {
	url := fmt.Sprintf("%s/Library/Refresh", s.cfg.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("jellyfin: building refresh request: %w", err)
	}
	req.Header.Set("X-Emby-Token", s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("jellyfin: refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("jellyfin: refresh returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// Test verifies that the Jellyfin server is reachable with the configured API key.
func (s *Server) Test(ctx context.Context) error {
	url := fmt.Sprintf("%s/System/Info", s.cfg.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("jellyfin: building test request: %w", err)
	}
	req.Header.Set("X-Emby-Token", s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("jellyfin: test request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("jellyfin: test returned %d: %s", resp.StatusCode, body)
	}
	return nil
}
