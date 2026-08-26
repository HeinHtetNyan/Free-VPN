// Package reports stores user-submitted issue reports — the main signal
// for "this ISP/carrier blocks the VPN" in Myanmar, which server-side
// metrics alone can't see (a connection that never even starts doesn't
// show up as a WireGuard peer at all). Same SQLite file as users.Store and
// servers.PeerStore, per docs/BACKEND.md.
package reports

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Report struct {
	ID          int64
	UserID      string
	Message     string
	IspName     string
	NetworkType string
	DeviceModel string
	OsVersion   string
	AppVersion  string
	CreatedAt   time.Time
	Delivered   bool
}

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		return nil, fmt.Errorf("configuring database: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			message TEXT NOT NULL,
			isp_name TEXT NOT NULL DEFAULT '',
			network_type TEXT NOT NULL DEFAULT '',
			device_model TEXT NOT NULL DEFAULT '',
			os_version TEXT NOT NULL DEFAULT '',
			app_version TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			delivered INTEGER NOT NULL DEFAULT 0
		);
	`); err != nil {
		return nil, fmt.Errorf("creating reports table: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// MaxMessageLen bounds what a client can submit — this is free text
// reaching an admin dashboard, not something to let grow unbounded.
const MaxMessageLen = 2000

func (s *Store) Create(r Report) error {
	if r.Message == "" {
		return fmt.Errorf("message is required")
	}
	if len(r.Message) > MaxMessageLen {
		r.Message = r.Message[:MaxMessageLen]
	}
	_, err := s.db.Exec(
		`INSERT INTO reports (user_id, message, isp_name, network_type, device_model, os_version, app_version)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.UserID, r.Message, r.IspName, r.NetworkType, r.DeviceModel, r.OsVersion, r.AppVersion,
	)
	if err != nil {
		return fmt.Errorf("inserting report: %w", err)
	}
	return nil
}

// Undelivered returns reports not yet successfully pushed to the admin
// dashboard, oldest first — picked up by internal/reporter's periodic loop.
// Capped at 200 per call so one huge backlog (e.g. the admin dashboard
// being down for a while) can't blow up a single push payload.
func (s *Store) Undelivered() ([]Report, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, message, isp_name, network_type, device_model, os_version, app_version, created_at
		FROM reports WHERE delivered = 0 ORDER BY created_at ASC LIMIT 200
	`)
	if err != nil {
		return nil, fmt.Errorf("querying undelivered reports: %w", err)
	}
	defer rows.Close()

	var out []Report
	for rows.Next() {
		var r Report
		if err := rows.Scan(&r.ID, &r.UserID, &r.Message, &r.IspName, &r.NetworkType, &r.DeviceModel, &r.OsVersion, &r.AppVersion, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning report: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkDelivered flags reports as successfully pushed, so future
// Undelivered() calls don't resend them.
func (s *Store) MarkDelivered(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`UPDATE reports SET delivered = 1 WHERE id = ?`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, id := range ids {
		if _, err := stmt.Exec(id); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
