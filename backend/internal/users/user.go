// Package users holds user records, backed by SQLite (modernc.org/sqlite —
// pure Go, no CGO, simplest thing that works on a single small VPS; see
// docs/BACKEND.md for why SQLite over Postgres at this scale).
package users

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Tier string

const (
	TierFree    Tier = "free"
	TierPremium Tier = "premium"
)

type User struct {
	ID        string
	DeviceID  string
	Token     string
	Tier      Tier
	CreatedAt time.Time
}

var ErrNotFound = errors.New("user not found")

type Store struct {
	db *sql.DB
}

// NewStore opens (creating if needed) the SQLite database at dbPath and
// ensures the schema exists. SQLite handles one writer at a time internally;
// max open connections is capped at 1 to avoid "database is locked" errors
// under concurrent writes rather than adding retry/backoff complexity that
// this traffic scale doesn't need.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL UNIQUE,
			token TEXT NOT NULL UNIQUE,
			tier TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);
	`); err != nil {
		return nil, fmt.Errorf("creating users table: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// GetOrCreateByDeviceID returns the existing user for deviceID, or creates
// one. Registration is idempotent by design: a reinstalled app resending the
// same locally-persisted device ID gets its existing account back, not a
// duplicate.
func (s *Store) GetOrCreateByDeviceID(deviceID string) (*User, error) {
	if u, err := s.getByDeviceID(deviceID); err == nil {
		return u, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	u := &User{
		ID:        generateID(),
		DeviceID:  deviceID,
		Token:     token,
		Tier:      TierFree,
		CreatedAt: time.Now(),
	}

	_, err = s.db.Exec(
		`INSERT INTO users (id, device_id, token, tier, created_at) VALUES (?, ?, ?, ?, ?)`,
		u.ID, u.DeviceID, u.Token, string(u.Tier), u.CreatedAt,
	)
	if err != nil {
		// Another request may have raced us to create the same device_id
		// between the SELECT above and this INSERT; fall back to reading
		// whatever won, rather than erroring the caller out.
		if existing, getErr := s.getByDeviceID(deviceID); getErr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return u, nil
}

func (s *Store) GetByToken(token string) (*User, error) {
	return s.scanUser(s.db.QueryRow(
		`SELECT id, device_id, token, tier, created_at FROM users WHERE token = ?`, token,
	))
}

func (s *Store) getByDeviceID(deviceID string) (*User, error) {
	return s.scanUser(s.db.QueryRow(
		`SELECT id, device_id, token, tier, created_at FROM users WHERE device_id = ?`, deviceID,
	))
}

func (s *Store) scanUser(row *sql.Row) (*User, error) {
	var u User
	var tier string
	if err := row.Scan(&u.ID, &u.DeviceID, &u.Token, &tier, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.Tier = Tier(tier)
	return &u, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
