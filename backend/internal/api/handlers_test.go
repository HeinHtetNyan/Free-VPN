package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"sy-vpn-backend/internal/servers"
	"sy-vpn-backend/internal/users"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	userStore, err := users.NewStore(dbPath)
	if err != nil {
		t.Fatalf("users.NewStore: %v", err)
	}
	t.Cleanup(func() { _ = userStore.Close() })

	peerStore, err := servers.NewPeerStore(dbPath, "10.66")
	if err != nil {
		t.Fatalf("servers.NewPeerStore: %v", err)
	}
	t.Cleanup(func() { _ = peerStore.Close() })

	locs := []servers.Location{
		{ID: "singapore", DisplayName: "Singapore", RelayAddress: "relay.example.com:51820"},
	}

	// WireGuard interface name intentionally bogus — RegisterPeer is
	// expected to fail cleanly in tests (no real interface exists here
	// either), matching the "best-effort" behavior documented in
	// docs/BACKEND.md.
	return NewServer(userStore, peerStore, locs, "wg-test-nonexistent", DefaultServerPublicKeyPlaceholder, servers.AmneziaParams{}, false, nil, nil, testAdminToken)
}

const testAdminToken = "test-admin-token"

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLocationsRequiresAuth(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/locations", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %d", rec.Code)
	}
}

func TestFullFlow_RegisterListConnect(t *testing.T) {
	srv := newTestServer(t)
	router := srv.Router()

	// Register.
	registerBody := strings.NewReader(`{"device_id":"test-device"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", registerBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var registerResp struct {
		Token string `json:"token"`
		Tier  string `json:"tier"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registerResp); err != nil {
		t.Fatalf("decoding register response: %v", err)
	}
	if registerResp.Token == "" || registerResp.Tier != "free" {
		t.Fatalf("unexpected register response: %+v", registerResp)
	}

	// Re-registering the same device ID should return the same token.
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"device_id":"test-device"}`)))
	var reRegisterResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(rec2.Body.Bytes(), &reRegisterResp)
	if reRegisterResp.Token != registerResp.Token {
		t.Fatalf("expected idempotent registration, got a different token")
	}

	// List locations (authenticated).
	locReq := httptest.NewRequest(http.MethodGet, "/locations", nil)
	locReq.Header.Set("Authorization", "Bearer "+registerResp.Token)
	locRec := httptest.NewRecorder()
	router.ServeHTTP(locRec, locReq)
	if locRec.Code != http.StatusOK {
		t.Fatalf("locations: expected 200, got %d: %s", locRec.Code, locRec.Body.String())
	}
	if !strings.Contains(locRec.Body.String(), "singapore") {
		t.Fatalf("expected locations response to contain singapore, got %s", locRec.Body.String())
	}

	// Connect to a known location.
	connectReq := httptest.NewRequest(http.MethodPost, "/connect", strings.NewReader(`{"location_id":"singapore"}`))
	connectReq.Header.Set("Authorization", "Bearer "+registerResp.Token)
	connectRec := httptest.NewRecorder()
	router.ServeHTTP(connectRec, connectReq)
	if connectRec.Code != http.StatusOK {
		t.Fatalf("connect: expected 200, got %d: %s", connectRec.Code, connectRec.Body.String())
	}
	var connectResp struct {
		Config    string `json:"config"`
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(connectRec.Body.Bytes(), &connectResp); err != nil {
		t.Fatalf("decoding connect response: %v", err)
	}
	if !strings.Contains(connectResp.Config, "[Interface]") || !strings.Contains(connectResp.Config, "relay.example.com:51820") {
		t.Fatalf("unexpected config content: %s", connectResp.Config)
	}

	// Connect to an unknown location should 404.
	badReq := httptest.NewRequest(http.MethodPost, "/connect", strings.NewReader(`{"location_id":"nowhere"}`))
	badReq.Header.Set("Authorization", "Bearer "+registerResp.Token)
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown location, got %d", badRec.Code)
	}
}

// TestConnect_Reconnect_RevokeIsBestEffort covers the fix for the ghost-peer
// pileup: reconnecting the same device now tries to revoke its previous
// peer (see revokeStalePeers), but that revoke runs against the test's
// deliberately bogus WireGuard interface and always fails — /connect must
// still succeed (best-effort, like RegisterPeer), and peer_allocations must
// still keep both historical rows (append-only, see PeerStore).
func TestConnect_Reconnect_RevokeIsBestEffort(t *testing.T) {
	srv := newTestServer(t)
	router := srv.Router()

	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"device_id":"reconnect-device"}`)))
	var registerResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(registerRec.Body.Bytes(), &registerResp)

	doConnect := func() string {
		req := httptest.NewRequest(http.MethodPost, "/connect", strings.NewReader(`{"location_id":"singapore"}`))
		req.Header.Set("Authorization", "Bearer "+registerResp.Token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("connect: expected 200 even though revoke of any previous peer fails against the test interface, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			PublicKey string `json:"public_key"`
			UserID    string `json:"user_id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decoding connect response: %v", err)
		}
		return resp.PublicKey
	}

	firstKey := doConnect()
	secondKey := doConnect()
	if firstKey == secondKey {
		t.Fatalf("expected a fresh keypair on reconnect, got the same public key twice")
	}

	// Find this user's ID via the peer rows themselves (handleConnect
	// doesn't return one in this response shape's earlier field name).
	peers, err := srv.Peers.AllPeers()
	if err != nil {
		t.Fatalf("AllPeers: %v", err)
	}
	var userID string
	var matched int
	for _, p := range peers {
		if p.PublicKey == firstKey || p.PublicKey == secondKey {
			matched++
			userID = p.UserID
		}
	}
	if matched != 2 {
		t.Fatalf("expected both connect calls' peers in peer_allocations, found %d", matched)
	}

	forUser, err := srv.Peers.PeersForUser(userID)
	if err != nil {
		t.Fatalf("PeersForUser: %v", err)
	}
	if len(forUser) != 2 {
		t.Fatalf("expected both historical peer rows to remain (append-only), got %d", len(forUser))
	}
}

func TestAdminFriends_RequiresAdminToken(t *testing.T) {
	srv := newTestServer(t)
	router := srv.Router()

	// No Authorization header at all.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/friends", strings.NewReader(`{"label":"alice"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no admin token, got %d", rec.Code)
	}

	// Wrong token.
	req := httptest.NewRequest(http.MethodPost, "/admin/friends", strings.NewReader(`{"label":"alice"}`))
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong admin token, got %d", rec2.Code)
	}
}

func TestAdminFriends_CreateIssuesConfig(t *testing.T) {
	srv := newTestServer(t)
	router := srv.Router()

	req := httptest.NewRequest(http.MethodPost, "/admin/friends", strings.NewReader(`{"label":"Alice Friend!"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create friend: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Config    string `json:"config"`
		PublicKey string `json:"public_key"`
		OwnerID   string `json:"owner_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding create-friend response: %v", err)
	}
	if !strings.Contains(resp.Config, "[Interface]") || resp.PublicKey == "" {
		t.Fatalf("unexpected create-friend response: %+v", resp)
	}
	if !strings.HasPrefix(resp.OwnerID, "friend:alice-friend-") {
		t.Fatalf("expected sanitized label in owner_id, got %q", resp.OwnerID)
	}

	// Revoking it goes through handleAdminRevokeFriend and calls
	// servers.RemovePeer against the (deliberately bogus) test interface —
	// this fails, and unlike RegisterPeer's best-effort handling in
	// issuePeerConfig, revoke is a hard failure by design: silently
	// reporting success when the peer wasn't actually removed from a real
	// interface would defeat the entire point of "delete = revoke access".
	// Only a real interface (the actual central server) can exercise the
	// success path — see infra/local-test for the project's usual way of
	// getting a real interface in a non-production environment.
	revokeReq := httptest.NewRequest(http.MethodPost, "/admin/friends/revoke", strings.NewReader(`{"public_key":"`+resp.PublicKey+`"}`))
	revokeReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	revokeRec := httptest.NewRecorder()
	router.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected revoke against a nonexistent test interface to fail with 500, got %d: %s", revokeRec.Code, revokeRec.Body.String())
	}
}
