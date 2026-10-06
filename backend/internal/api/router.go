package api

import (
	"net/http"
	"time"

	"sy-vpn-backend/internal/auth"
)

// Router wires up all HTTP routes. Handed a plain *http.ServeMux rather than
// a framework — the route count is small enough that stdlib routing (with
// Go 1.22+'s method-aware patterns) is simpler than adding a dependency.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	// Per-user limits (created per Router so each server/test has its own state).
	// /connect: 30/min burst 30 is far above normal reconnect behaviour.
	// /report: burst 5, then 1 per 12s (5/min sustained), plenty for a human.
	limitConnect := newUserLimiter(30, 2*time.Second)
	limitReport := newUserLimiter(5, 12*time.Second)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /privacy", s.handlePrivacy)

	mux.HandleFunc("POST /auth/register", RateLimitRegister(s.handleRegister))
	mux.HandleFunc("GET /locations", auth.Require(s.Users, s.handleListLocations))
	mux.HandleFunc("POST /connect", auth.Require(s.Users, limitConnect.wrap(s.handleConnect)))
	mux.HandleFunc("GET /stats", auth.Require(s.Users, s.handleStats))
	mux.HandleFunc("POST /report", auth.Require(s.Users, limitReport.wrap(s.handleReport)))

	// Admin-only: called by the Activation-Licenses admin backend, never by
	// the app itself — see internal/auth.RequireAdminToken and
	// handleAdminCreateFriend/handleAdminRevokeFriend.
	mux.HandleFunc("POST /admin/friends", auth.RequireAdminToken(s.AdminToken, s.handleAdminCreateFriend))
	mux.HandleFunc("POST /admin/friends/revoke", auth.RequireAdminToken(s.AdminToken, s.handleAdminRevokeFriend))
	mux.HandleFunc("GET /admin/locations", auth.RequireAdminToken(s.AdminToken, s.handleAdminListLocations))

	return mux
}
