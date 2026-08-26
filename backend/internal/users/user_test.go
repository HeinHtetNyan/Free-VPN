package users

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestGetOrCreateByDeviceID_IsIdempotent(t *testing.T) {
	s := newTestStore(t)

	first, err := s.GetOrCreateByDeviceID("device-a")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	second, err := s.GetOrCreateByDeviceID("device-a")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if first.ID != second.ID || first.Token != second.Token {
		t.Fatalf("expected same user/token, got %+v vs %+v", first, second)
	}
}

func TestGetOrCreateByDeviceID_DifferentDevicesGetDifferentUsers(t *testing.T) {
	s := newTestStore(t)

	a, _ := s.GetOrCreateByDeviceID("device-a")
	b, _ := s.GetOrCreateByDeviceID("device-b")

	if a.ID == b.ID || a.Token == b.Token {
		t.Fatalf("expected distinct users, got same: %+v", a)
	}
}

func TestGetByToken(t *testing.T) {
	s := newTestStore(t)

	created, _ := s.GetOrCreateByDeviceID("device-a")

	found, err := s.GetByToken(created.Token)
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("expected id %s, got %s", created.ID, found.ID)
	}

	if _, err := s.GetByToken("not-a-real-token"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestNewStore_PersistsAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	s1, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	u, _ := s1.GetOrCreateByDeviceID("device-a")
	_ = s1.Close()

	s2, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("reopening NewStore: %v", err)
	}
	defer s2.Close()

	found, err := s2.GetByToken(u.Token)
	if err != nil {
		t.Fatalf("GetByToken after reopen: %v", err)
	}
	if found.ID != u.ID {
		t.Fatalf("expected id %s after reopen, got %s", u.ID, found.ID)
	}
}
