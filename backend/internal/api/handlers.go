package api

import (
	"encoding/json"
	"log"
	"net/http"

	"sy-vpn-backend/internal/auth"
	"sy-vpn-backend/internal/reports"
	"sy-vpn-backend/internal/servers"
	"sy-vpn-backend/internal/stats"
	"sy-vpn-backend/internal/users"
)

// DefaultServerPublicKeyPlaceholder is what ServerPublicKey defaults to when
// unset — configs built with it are structurally correct but won't connect
// to anything. Set the real SERVER_PUBLIC_KEY env var (see cmd/server/main.go)
// once infra/scripts/setup-central-server.sh has been run against a real
// server and printed the real key (see docs/OPEN_QUESTIONS.md).
const DefaultServerPublicKeyPlaceholder = "REPLACE_WITH_CENTRAL_SERVER_PUBLIC_KEY"

type Server struct {
	Users              *users.Store
	Peers              *servers.PeerStore
	Locations          []servers.Location
	WireGuardIfaceName string
	ServerPublicKey    string
	// Stats is nil in tests/environments that don't wire one up (see
	// cmd/server/main.go) — handleStats degrades to a zero count rather than
	// panicking or erroring, since "no data yet" is a normal startup state.
	Stats   *stats.Collector
	Reports *reports.Store
}

func NewServer(userStore *users.Store, peerStore *servers.PeerStore, locations []servers.Location, wgIfaceName, serverPublicKey string, statsCollector *stats.Collector, reportStore *reports.Store) *Server {
	return &Server{Users: userStore, Peers: peerStore, Locations: locations, WireGuardIfaceName: wgIfaceName, ServerPublicKey: serverPublicKey, Stats: statsCollector, Reports: reportStore}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// handleRegister implements anonymous, device-bound registration: the app
// generates and persists its own random device ID locally on first launch
// and sends it here. Idempotent — the same device ID always resolves to the
// same account (see internal/users.Store.GetOrCreateByDeviceID and
// docs/DECISIONS.md for why this over email/password accounts).
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	u, err := s.Users.GetOrCreateByDeviceID(body.DeviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not register device")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"token": u.Token,
		"tier":  string(u.Tier),
	})
}

// handleListLocations returns the app's location picker data. Deliberately
// does not expose relay_address/central_server internals to the client —
// those are resolved server-side in handleConnect.
func (s *Server) handleListLocations(w http.ResponseWriter, r *http.Request) {
	type locationView struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		// RelayHost (no port) lets the app measure its own latency to this
		// location — see servers.Location.RelayHost.
		RelayHost string `json:"relay_host"`
	}
	views := make([]locationView, 0, len(s.Locations))
	for _, l := range s.Locations {
		views = append(views, locationView{ID: l.ID, DisplayName: l.DisplayName, RelayHost: l.RelayHost()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"locations": views})
}

// handleConnect issues a fresh WireGuard client config for the requested
// location. See docs/ARCHITECTURE.md: the config's Endpoint is a LocalToNet
// relay, and the placeholder server public key means this won't actually
// connect to anything until a real central server exists.
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var body struct {
		LocationID string `json:"location_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.LocationID == "" {
		writeError(w, http.StatusBadRequest, "location_id is required")
		return
	}

	loc, ok := servers.FindLocation(s.Locations, body.LocationID)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown location_id")
		return
	}

	keys, err := servers.GenerateKeyPair()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate keys")
		return
	}

	assignedIP, err := s.Peers.AllocateIP(keys.PublicKey, user.ID, loc.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not allocate tunnel address")
		return
	}

	// Best-effort: register the peer on the live WireGuard interface so it
	// can actually connect. This fails cleanly (logged, not returned as an
	// HTTP error) everywhere except the real central server, since no such
	// interface exists yet — see docs/OPEN_QUESTIONS.md. Once a central
	// server is deployed for real, promote this to a hard failure: a config
	// that was never registered as a peer will never connect, and silently
	// handing it out at that point would just be confusing.
	if err := servers.RegisterPeer(s.WireGuardIfaceName, keys.PublicKey, assignedIP); err != nil {
		log.Printf("warning: could not register peer on WireGuard interface %q (expected until a central server is deployed): %v", s.WireGuardIfaceName, err)
	}

	config := servers.BuildClientConfig(loc, keys, s.ServerPublicKey, assignedIP)

	writeJSON(w, http.StatusOK, map[string]any{
		"location_id": loc.ID,
		"config":      config,
		"public_key":  keys.PublicKey,
		"user_id":     user.ID,
	})
}

// handleReport lets a user submit a free-text issue report (e.g. "MPT
// blocks this VPN"), with client-supplied technical context attached —
// this is the main signal for ISP-level blocking in Myanmar, which
// server-side connection metrics alone can't see: a connection an ISP
// blocks outright never even shows up as a WireGuard peer.
func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var body struct {
		Message     string `json:"message"`
		IspName     string `json:"isp_name"`
		NetworkType string `json:"network_type"`
		DeviceModel string `json:"device_model"`
		OsVersion   string `json:"os_version"`
		AppVersion  string `json:"app_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	if s.Reports == nil {
		writeError(w, http.StatusServiceUnavailable, "reports not available")
		return
	}

	err := s.Reports.Create(reports.Report{
		UserID: user.ID, Message: body.Message, IspName: body.IspName, NetworkType: body.NetworkType,
		DeviceModel: body.DeviceModel, OsVersion: body.OsVersion, AppVersion: body.AppVersion,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save report")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleStats is deliberately aggregate-only — a per-user breakdown belongs
// in the admin dashboard (a separate, access-controlled project), not a
// public-ish endpoint every registered device can call.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	connectedNow := 0
	if s.Stats != nil {
		connectedNow = s.Stats.Current().ConnectedNow
	}
	writeJSON(w, http.StatusOK, map[string]any{"connected_now": connectedNow})
}
