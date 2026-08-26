package api

import (
	"net/http"

	"sy-vpn-backend/internal/auth"
)

// Router wires up all HTTP routes. Handed a plain *http.ServeMux rather than
// a framework — the route count is small enough that stdlib routing (with
// Go 1.22+'s method-aware patterns) is simpler than adding a dependency.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /auth/register", s.handleRegister)
	mux.HandleFunc("GET /locations", auth.Require(s.Users, s.handleListLocations))
	mux.HandleFunc("POST /connect", auth.Require(s.Users, s.handleConnect))
	mux.HandleFunc("GET /stats", auth.Require(s.Users, s.handleStats))
	mux.HandleFunc("POST /report", auth.Require(s.Users, s.handleReport))

	return mux
}
