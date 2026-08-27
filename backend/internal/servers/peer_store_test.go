package servers

import (
	"path/filepath"
	"testing"
)

func newTestPeerStore(t *testing.T) *PeerStore {
	t.Helper()
	s, err := NewPeerStore(filepath.Join(t.TempDir(), "test.db"), "10.66")
	if err != nil {
		t.Fatalf("NewPeerStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestAllocateIP_IsIdempotentPerKey(t *testing.T) {
	s := newTestPeerStore(t)

	first, err := s.AllocateIP("pubkey-a", "user-1", "loc-1")
	if err != nil {
		t.Fatalf("first allocation: %v", err)
	}

	second, err := s.AllocateIP("pubkey-a", "user-1", "loc-1")
	if err != nil {
		t.Fatalf("second allocation: %v", err)
	}

	if first != second {
		t.Fatalf("expected same IP for repeated allocation, got %s vs %s", first, second)
	}
}

func TestAllocateIP_DistinctKeysGetDistinctIPs(t *testing.T) {
	s := newTestPeerStore(t)

	a, _ := s.AllocateIP("pubkey-a", "user-1", "loc-1")
	b, _ := s.AllocateIP("pubkey-b", "user-1", "loc-1")

	if a == b {
		t.Fatalf("expected distinct IPs, both got %s", a)
	}
}

func TestAllocateIP_UsesConfiguredSubnetBase(t *testing.T) {
	s, err := NewPeerStore(filepath.Join(t.TempDir(), "test.db"), "10.67")
	if err != nil {
		t.Fatalf("NewPeerStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	ip, err := s.AllocateIP("pubkey-a", "user-1", "loc-1")
	if err != nil {
		t.Fatalf("AllocateIP: %v", err)
	}
	if got, want := ip[:5], "10.67"; got != want {
		t.Fatalf("expected an IP in the configured 10.67 subnet, got %s", ip)
	}
}

func TestAllocateIP_EmptySubnetBaseDefaultsTo10_66(t *testing.T) {
	s, err := NewPeerStore(filepath.Join(t.TempDir(), "test.db"), "")
	if err != nil {
		t.Fatalf("NewPeerStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	ip, err := s.AllocateIP("pubkey-a", "user-1", "loc-1")
	if err != nil {
		t.Fatalf("AllocateIP: %v", err)
	}
	if got, want := ip[:5], "10.66"; got != want {
		t.Fatalf("expected the default 10.66 subnet, got %s", ip)
	}
}

func TestPeersForUser_ReturnsOnlyThatUsersPeersOldestFirst(t *testing.T) {
	s := newTestPeerStore(t)

	first, _ := s.AllocateIP("pubkey-a", "user-1", "loc-1")
	second, _ := s.AllocateIP("pubkey-b", "user-1", "loc-1")
	_, _ = s.AllocateIP("pubkey-c", "user-2", "loc-1")

	got, err := s.PeersForUser("user-1")
	if err != nil {
		t.Fatalf("PeersForUser: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 peers for user-1, got %d: %+v", len(got), got)
	}
	if got[0].AssignedIP != first || got[1].AssignedIP != second {
		t.Fatalf("expected oldest-first order [%s, %s], got [%s, %s]", first, second, got[0].AssignedIP, got[1].AssignedIP)
	}
	for _, p := range got {
		if p.UserID != "user-1" {
			t.Fatalf("expected only user-1's peers, got one for %q", p.UserID)
		}
	}
}

func TestPeersForUser_UnknownUserReturnsEmpty(t *testing.T) {
	s := newTestPeerStore(t)
	_, _ = s.AllocateIP("pubkey-a", "user-1", "loc-1")

	got, err := s.PeersForUser("nobody")
	if err != nil {
		t.Fatalf("PeersForUser: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no peers for unknown user, got %d", len(got))
	}
}

func TestLoadLocations(t *testing.T) {
	locs, err := LoadLocations()
	if err != nil {
		t.Fatalf("LoadLocations: %v", err)
	}
	if len(locs) == 0 {
		t.Fatal("expected at least one location")
	}

	if _, ok := FindLocation(locs, locs[0].ID); !ok {
		t.Fatalf("FindLocation could not find %q which LoadLocations just returned", locs[0].ID)
	}
	if _, ok := FindLocation(locs, "definitely-not-a-real-location"); ok {
		t.Fatal("FindLocation should not find a nonexistent location")
	}
}
