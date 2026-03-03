package middleware

import (
	"net/http"

	"github.com/davidfic/luminarr/internal/config"
)

const apiKeyHeader = "X-Api-Key" //nolint:gosec // G101 false positive: this is the header name, not a credential value

// Auth returns a middleware that enforces API key authentication.
// Requests without a valid X-Api-Key header receive a 401 response.
// The key value is never logged.
func Auth(apiKey config.Secret) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := r.Header.Get(apiKeyHeader)
			if provided == "" || provided != apiKey.Value() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("WWW-Authenticate", `ApiKey realm="Luminarr"`)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"title":"Unauthorized","status":401,"detail":"A valid X-Api-Key header is required."}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
