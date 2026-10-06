package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func registerToken(t *testing.T, srv *Server, deviceID string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"device_id":"`+deviceID+`"}`))
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register %q: %d %s", deviceID, rec.Code, rec.Body.String())
	}
	i := strings.Index(rec.Body.String(), `"token":"`) + 9
	rest := rec.Body.String()[i:]
	return rest[:strings.Index(rest, `"`)]
}

func TestRegisterDeviceIDValidation(t *testing.T) {
	srv := newTestServer(t)
	// Formats the real app / existing users have: UUID, base64url-22, test ids.
	for _, ok := range []string{"123e4567-e89b-12d3-a456-426614174000", "5_jIv9lYO3HFGP_KabqAXw", "bd-810wVtvOOxy8bjVgV0g", "test-device"} {
		registerToken(t, srv, ok)
	}
	bad := []string{"short", strings.Repeat("a", 129), "has space here", "semi;colon;;", "../../etc/passwd"}
	for _, b := range bad {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"device_id":"`+b+`"}`))
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("device_id %q: want 400 got %d", b, rec.Code)
		}
	}
}

func TestBodyTooLargeRejected(t *testing.T) {
	srv := newTestServer(t)
	big := `{"device_id":"` + strings.Repeat("a", 70*1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(big))
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for oversized body, got %d", rec.Code)
	}
}

func TestReportLengthCapsAndRateLimit(t *testing.T) {
	srv := newTestServer(t)
	tok := registerToken(t, srv, "report-device-1")
	post := func(router http.Handler, body string) int {
		req := httptest.NewRequest(http.MethodPost, "/report", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	router := srv.Router()
	// Reports store is nil in tests: valid report reaches 503, which is past validation.
	if c := post(router, `{"message":"`+strings.Repeat("m", 2001)+`"}`); c != http.StatusBadRequest {
		t.Errorf("long message: want 400 got %d", c)
	}
	if c := post(router, `{"message":"hi","isp_name":"`+strings.Repeat("i", 101)+`"}`); c != http.StatusBadRequest {
		t.Errorf("long isp: want 400 got %d", c)
	}
	if c := post(router, `{"message":"`+strings.Repeat("m", 2000)+`","isp_name":"`+strings.Repeat("i", 100)+`"}`); c != http.StatusServiceUnavailable {
		t.Errorf("max-size ok report: want 503 (validation passed) got %d", c)
	}
	// Burst of 5 total allowed; 3 used so far, so 2 more pass then 429.
	got429 := false
	for i := 0; i < 10; i++ {
		if post(router, `{"message":"hi"}`) == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Error("expected 429 after report burst")
	}
}

func TestConnectLimiterGenerousButBounded(t *testing.T) {
	srv := newTestServer(t)
	router := srv.Router()
	tok := registerToken(t, srv, "connect-device-1")
	other := registerToken(t, srv, "connect-device-2")
	do := func(token string) int {
		req := httptest.NewRequest(http.MethodPost, "/connect", strings.NewReader(`{"location_id":"singapore"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	for i := 0; i < 30; i++ {
		if c := do(tok); c != http.StatusOK {
			t.Fatalf("connect #%d should succeed, got %d", i+1, c)
		}
	}
	if c := do(tok); c != http.StatusTooManyRequests {
		t.Fatalf("31st rapid connect: want 429 got %d", c)
	}
	if c := do(other); c != http.StatusOK {
		t.Fatalf("other user must be unaffected, got %d", c)
	}
}

func TestClientIPTrust(t *testing.T) {
	mk := func(remote, cf, xff string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = remote
		if cf != "" {
			r.Header.Set("Cf-Connecting-Ip", cf)
		}
		if xff != "" {
			r.Header.Set("X-Forwarded-For", xff)
		}
		return r
	}
	cases := []struct {
		name string
		r    *http.Request
		want string
	}{
		{"loopback trusts cf", mk("127.0.0.1:5555", "203.0.113.9", ""), "203.0.113.9"},
		{"loopback trusts xff", mk("127.0.0.1:5555", "", "198.51.100.4, 10.0.0.1"), "198.51.100.4"},
		{"private trusts cf", mk("172.18.0.2:1", "203.0.113.9", ""), "203.0.113.9"},
		{"ipv6 loopback", mk("[::1]:80", "203.0.113.9", ""), "203.0.113.9"},
		{"public peer ignores spoofed cf", mk("198.51.100.77:4444", "1.2.3.4", "5.6.7.8"), "198.51.100.77"},
		{"loopback no headers", mk("127.0.0.1:5555", "", ""), "127.0.0.1"},
	}
	for _, c := range cases {
		if got := clientIP(c.r); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}
