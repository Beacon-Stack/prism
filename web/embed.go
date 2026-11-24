// Package web serves the embedded Prism web UI.
// Static assets live under web/static/ and are compiled into the binary
// via the //go:embed directive.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed static
var staticFiles embed.FS

// ServeStatic returns an http.Handler that serves the React SPA.
//
// Request routing:
//   - /assets/*  → cached JS/CSS bundles from static/assets/
//   - Any path that resolves to an existing file → serve it
//   - Everything else → serve index.html (React Router handles client-side routing)
func ServeStatic() http.Handler {
	raw, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		panic("web: could not read embedded index.html: " + err.Error())
	}
	indexHTML := raw

	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic("web: could not sub embedded static FS: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Normalise path.
		p := path.Clean("/" + r.URL.Path)

		// Assets are versioned hashes — cache aggressively.
		if strings.HasPrefix(p, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check whether a static file exists at this path.
		// Avoid serving index.html for directory requests.
		if p != "/" && p != "/index.html" {
			f, err := sub.Open(strings.TrimPrefix(p, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fall back to the SPA shell for all unknown paths.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(indexHTML)
	})
}
