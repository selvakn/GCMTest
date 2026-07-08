package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// RequireOperatorToken wraps a handler so it only runs for requests bearing
// a valid "Authorization: Bearer <token>" header matching the configured
// operator token (FR-029). Used to restrict round-triggering to authorized
// callers; device enrollment intentionally does not use this middleware
// (spec.md §8 — enrollment is open fleet membership, not privileged).
func RequireOperatorToken(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, prefix) {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		provided := strings.TrimPrefix(auth, prefix)
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return
		}
		next(w, r)
	}
}
