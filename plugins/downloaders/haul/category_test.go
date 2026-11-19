package haul

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beacon-stack/prism/pkg/plugin"
)

// With no category configured, Prism must still send "movies" so Haul
// files the download under <download dir>/movies on a fresh install.
func TestAdd_DefaultsCategoryToMovies(t *testing.T) {
	for _, configured := range []string{"", "  ", "anime"} {
		var got string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			got, _ = body["category"].(string)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"info_hash":"abc"}`))
		}))
		c := New(Config{URL: srv.URL, APIKey: "k", Category: configured})
		_, _ = c.Add(context.Background(), plugin.Release{Title: "T", DownloadURL: "magnet:?xt=urn:btih:abc"})
		srv.Close()

		want := configured
		if strings.TrimSpace(configured) == "" {
			want = DefaultCategory
		}
		if got != want {
			t.Errorf("configured %q: sent category %q, want %q", configured, got, want)
		}
	}
}
