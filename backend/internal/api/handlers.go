package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

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
	// AmneziaEnabled reflects whether AWG_* env vars were set (see
	// servers.AmneziaParamsFromEnv) — false means plain WireGuard, matching
	// every deployment before docs/ARCHITECTURE.md's "Censorship resistance"
	// work. AmneziaParams is the zero value when AmneziaEnabled is false.
	AmneziaEnabled bool
	AmneziaParams  servers.AmneziaParams
	// Stats is nil in tests/environments that don't wire one up (see
	// cmd/server/main.go) — handleStats degrades to a zero count rather than
	// panicking or erroring, since "no data yet" is a normal startup state.
	Stats   *stats.Collector
	Reports *reports.Store
	// AdminToken gates the /admin/friends routes (see internal/auth.RequireAdminToken)
	// — empty means those routes refuse every request. Set from VPN_ADMIN_TOKEN,
	// shared only with the Activation-Licenses admin backend.
	AdminToken string
}

func NewServer(userStore *users.Store, peerStore *servers.PeerStore, locations []servers.Location, wgIfaceName, serverPublicKey string, amneziaParams servers.AmneziaParams, amneziaEnabled bool, statsCollector *stats.Collector, reportStore *reports.Store, adminToken string) *Server {
	return &Server{
		Users: userStore, Peers: peerStore, Locations: locations, WireGuardIfaceName: wgIfaceName, ServerPublicKey: serverPublicKey,
		AmneziaParams: amneziaParams, AmneziaEnabled: amneziaEnabled,
		Stats: statsCollector, Reports: reportStore, AdminToken: adminToken,
	}
}

// issuePeerConfig generates a fresh keypair for ownerID at loc, registers it
// on the live WireGuard interface, and renders its client config — the
// shared core of handleConnect (real app users) and handleAdminCreateFriend
// (manually-issued friend/beta-tester configs, see docs/OPEN_QUESTIONS.md).
func (s *Server) issuePeerConfig(loc servers.Location, ownerID string) (config, publicKey string, err error) {
	keys, err := servers.GenerateKeyPair()
	if err != nil {
		return "", "", fmt.Errorf("could not generate keys: %w", err)
	}

	assignedIP, err := s.Peers.AllocateIP(keys.PublicKey, ownerID, loc.ID)
	if err != nil {
		return "", "", fmt.Errorf("could not allocate tunnel address: %w", err)
	}

	if err := servers.RegisterPeer(s.WireGuardIfaceName, keys.PublicKey, assignedIP, s.AmneziaEnabled); err != nil {
		log.Printf("warning: could not register peer on WireGuard interface %q (expected until a central server is deployed): %v", s.WireGuardIfaceName, err)
	}

	s.revokeStalePeers(ownerID, keys.PublicKey)

	config = servers.BuildClientConfig(loc, keys, s.ServerPublicKey, assignedIP, s.AmneziaParams, s.AmneziaEnabled)
	return config, keys.PublicKey, nil
}

// revokeStalePeers drops every WireGuard peer previously issued to ownerID
// other than currentPublicKey — the app never keeps an old config around
// past its next /connect call, so once a fresh peer is live, any earlier
// ones for the same owner are just dead weight left registered on the
// interface (see docs/DECISIONS.md: this is what filled the admin dashboard
// with dozens of never-connected ghost peers). Skips "friend:" owners:
// those are manually admin-issued, one config per issuance, and an admin
// may reissue the same label for a different device — reusing ownerID
// there would risk revoking a friend's still-in-use config out from under
// them, so friends stay purely admin-managed via /admin/friends/revoke.
// Best-effort like RegisterPeer above: a lookup or revoke failure here
// doesn't fail the /connect request, since the new peer already works
// regardless.
func (s *Server) revokeStalePeers(ownerID, currentPublicKey string) {
	if ownerID == "" || strings.HasPrefix(ownerID, "friend:") {
		return
	}
	previous, err := s.Peers.PeersForUser(ownerID)
	if err != nil {
		log.Printf("warning: could not list previous peers for %s: %v", ownerID, err)
		return
	}
	for _, p := range previous {
		if p.PublicKey == currentPublicKey {
			continue
		}
		if err := servers.RemovePeer(s.WireGuardIfaceName, p.PublicKey); err != nil {
			log.Printf("warning: could not revoke stale peer %s for %s: %v", p.PublicKey, ownerID, err)
		}
	}
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

	config, publicKey, err := s.issuePeerConfig(loc, user.ID)
	if err != nil {
		log.Printf("issuing peer config for user %s: %v", user.ID, err)
		writeError(w, http.StatusInternalServerError, "could not issue config")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"location_id": loc.ID,
		"config":      config,
		"public_key":  publicKey,
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

// handleAdminListLocations lets the Activation-Licenses admin backend show a
// location picker when issuing a friend config — handleListLocations already
// does this for the app itself, but that's gated on a real device/user
// token (auth.Require), which the admin backend doesn't have; this is the
// same data behind the admin token instead. Omits relay_host (unlike
// handleListLocations) since the admin picker has no latency measurement to
// do with it.
func (s *Server) handleAdminListLocations(w http.ResponseWriter, r *http.Request) {
	type locationView struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	views := make([]locationView, 0, len(s.Locations))
	for _, l := range s.Locations {
		views = append(views, locationView{ID: l.ID, DisplayName: l.DisplayName})
	}
	writeJSON(w, http.StatusOK, map[string]any{"locations": views})
}

// handleAdminCreateFriend issues a config the same way handleConnect does,
// but for someone who never goes through the app's own /auth/register —
// manually-shared configs for friends/beta testers (see
// infra/scripts/generate-friend-config.sh, which this replaces once the
// Activation-Licenses "VPN Friends" tab exists). ownerID is a synthetic
// "friend:<label>-<random>" string, not a real users.Store row — friends
// never authenticate against this backend themselves, so there's nothing
// for a users row to represent.
func (s *Server) handleAdminCreateFriend(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Label      string `json:"label"`
		LocationID string `json:"location_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var loc servers.Location
	if body.LocationID != "" {
		var ok bool
		loc, ok = servers.FindLocation(s.Locations, body.LocationID)
		if !ok {
			writeError(w, http.StatusNotFound, "unknown location_id")
			return
		}
	} else if len(s.Locations) > 0 {
		loc = s.Locations[0]
	} else {
		writeError(w, http.StatusServiceUnavailable, "no locations configured")
		return
	}

	suffix, err := randomHex(4)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate id")
		return
	}
	ownerID := "friend:" + sanitizeLabel(body.Label) + "-" + suffix

	config, publicKey, err := s.issuePeerConfig(loc, ownerID)
	if err != nil {
		log.Printf("issuing friend peer config (owner %s): %v", ownerID, err)
		writeError(w, http.StatusInternalServerError, "could not issue config")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"location_id": loc.ID,
		"config":      config,
		"public_key":  publicKey,
		"owner_id":    ownerID,
	})
}

// handleAdminRevokeFriend drops a previously-issued peer from the live
// WireGuard interface so its config immediately stops working. Takes the
// public key in a JSON body rather than a URL path segment — WireGuard
// public keys are standard base64 and can contain '/', which would collide
// with path routing.
func (s *Server) handleAdminRevokeFriend(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PublicKey == "" {
		writeError(w, http.StatusBadRequest, "public_key is required")
		return
	}

	if err := servers.RemovePeer(s.WireGuardIfaceName, body.PublicKey); err != nil {
		log.Printf("revoking peer %s on %q: %v", body.PublicKey, s.WireGuardIfaceName, err)
		writeError(w, http.StatusInternalServerError, "could not revoke peer")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random suffix: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// sanitizeLabel keeps only characters safe to embed in a peer_allocations
// user_id / device_id string, so a friend's display name can't inject
// anything unexpected into storage or logs. Falls back to "friend" if
// nothing safe survives.
func sanitizeLabel(label string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(label) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "friend"
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}
