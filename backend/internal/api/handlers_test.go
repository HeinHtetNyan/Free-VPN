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

	peerStore, err := servers.NewPeerStore(dbPath)
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
	return NewServer(userStore, peerStore, locs, "wg-test-nonexistent", DefaultServerPublicKeyPlaceholder, nil, nil)
}

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
