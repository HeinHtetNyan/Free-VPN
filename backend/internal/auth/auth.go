// Package auth provides the HTTP middleware that resolves a Bearer token to
// a user. Identity itself (device-ID registration) lives in internal/users —
// this package only deals with the request-time "who is this" question.
package auth

import (
	"context"
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
