// Package auth provides the HTTP middleware that resolves a Bearer token to
// a user. Identity itself (device-ID registration) lives in internal/users —
// this package only deals with the request-time "who is this" question.
package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"sy-vpn-backend/internal/users"
)

type contextKey int

const userContextKey contextKey = 0

// Require wraps a handler so it only runs for requests carrying a valid
// "Authorization: Bearer <token>" header, and makes the resolved user
// available via UserFromContext.
func Require(store *users.Store, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(authHeader, "Bearer ")
		if !ok || token == "" {
			http.Error(w, "missing or malformed Authorization header", http.StatusUnauthorized)
			return
		}

		user, err := store.GetByToken(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

func UserFromContext(ctx context.Context) (*users.User, bool) {
	u, ok := ctx.Value(userContextKey).(*users.User)
	return u, ok
}

// RequireAdminToken wraps a handler so it only runs for requests carrying
// "Authorization: Bearer <adminToken>" — a single shared secret (like
// VPN_INGEST_TOKEN's ingest side, see docs/BACKEND.md), not a per-caller
// credential, since only the Activation-Licenses admin backend calls these
// routes. If adminToken is unset, every request is refused rather than
// silently accepting anything.
func RequireAdminToken(adminToken string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminToken == "" {
			http.Error(w, "admin endpoints not configured", http.StatusServiceUnavailable)
			return
		}
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) != 1 {
			http.Error(w, "missing or invalid admin token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
