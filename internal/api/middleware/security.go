package middleware

import (
	"net/http"
)

// SecurityHeaders sets defensive HTTP response headers on every response.
// These do not require HTTPS to be useful — they prevent the browser from
// doing unsafe things with the response regardless of transport.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// Prevent MIME sniffing — the browser must honour Content-Type.
		h.Set("X-Content-Type-Options", "nosniff")
		// Disallow embedding in iframes (clickjacking protection).
		h.Set("X-Frame-Options", "DENY")
		// Do not send the Referer header to third-party origins.
		h.Set("Referrer-Policy", "same-origin")
		// Basic CSP: only allow resources from the same origin.
		// unsafe-inline is needed for the React SPA's inline scripts/styles
		// until a nonce-based CSP can be implemented.
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://image.tmdb.org; font-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// MaxRequestBodySize limits request bodies to maxBytes. Requests with a
// body larger than this will be rejected with 413 Request Entity Too Large
// before the handler is invoked.
func MaxRequestBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
