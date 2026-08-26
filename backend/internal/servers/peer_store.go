package servers

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// PeerStore tracks WireGuard client IP assignments in the same SQLite file
// users.Store uses, so both are on one durable DB file per docs/BACKEND.md.
// IP allocation here is a simple monotonically-increasing sequence — it does
// not reclaim IPs from peers that are later removed. Fine at the scale this
// is starting at; revisit (e.g. reuse freed IPs) if the 10.66.0.0/16 pool
// (~65k addresses) ever becomes a real constraint.
type PeerStore struct {
	db *sql.DB
}

func NewPeerStore(dbPath string) (*PeerStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		return nil, fmt.Errorf("configuring database: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS peer_allocations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			public_key TEXT NOT NULL UNIQUE,
			assigned_ip TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return nil, fmt.Errorf("creating peer_allocations table: %w", err)
	}

	return &PeerStore{db: db}, nil
}

func (p *PeerStore) Close() error {
	return p.db.Close()
}

// AllocateIP assigns (or returns the existing) tunnel-internal IP for
// publicKey, out of 10.66.0.0/16. .0 and .1 in each /24 are skipped
// (network address and a reserved slot for the server itself).
func (p *PeerStore) AllocateIP(publicKey string) (string, error) {
	var existing string
	err := p.db.QueryRow(`SELECT assigned_ip FROM peer_allocations WHERE public_key = ?`, publicKey).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("looking up existing allocation: %w", err)
	}

	res, err := p.db.Exec(`INSERT INTO peer_allocations (public_key, assigned_ip) VALUES (?, '')`, publicKey)
	if err != nil {
		return "", fmt.Errorf("inserting allocation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return "", fmt.Errorf("reading allocation id: %w", err)
	}

	ip := sequenceToIP(id)
	if _, err := p.db.Exec(`UPDATE peer_allocations SET assigned_ip = ? WHERE id = ?`, ip, id); err != nil {
		return "", fmt.Errorf("saving assigned ip: %w", err)
	}
	return ip, nil
}

func sequenceToIP(seq int64) string {
	offset := seq + 1 // start at .2, not .0/.1
	octet3 := (offset / 254) % 256
	octet4 := (offset % 254) + 2
	return fmt.Sprintf("10.66.%d.%d", octet3, octet4)
}
