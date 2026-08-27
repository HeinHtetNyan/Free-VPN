package servers

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// PeerStore tracks WireGuard client IP assignments in the same SQLite file
// users.Store uses, so both are on one durable DB file per docs/BACKEND.md.
// IP allocation here is a simple monotonically-increasing sequence — it does
// not reclaim IPs from peers that are later removed. Fine at the scale this
// is starting at; revisit (e.g. reuse freed IPs) if the pool (~65k
// addresses) ever becomes a real constraint.
type PeerStore struct {
	db *sql.DB
	// subnetBase is the "A.B" prefix of the /16 assignedIP comes from — must
	// match whatever WG_INTERFACE's own `ip addr` subnet actually is (10.66
	// for wg0, 10.67 for awg0 — see infra/scripts/setup-central-server.sh /
	// setup-amnezia-interface.sh). Get this wrong and AllocateIP hands out
	// addresses outside the interface's routed range — see docs/DECISIONS.md
	// 2026-08-27: this shipped hardcoded to "10.66" even after awg0
	// (10.67.0.0/16) went live, so a real client got assigned 10.66.0.33 on
	// an interface that only routes 10.67.0.0/16.
	subnetBase string
}

func NewPeerStore(dbPath, subnetBase string) (*PeerStore, error) {
	if subnetBase == "" {
		subnetBase = "10.66"
	}
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

	// Added later than the table itself (see docs/DECISIONS.md) — original
	// rows predate user/location attribution, so these are nullable rather
	// than a destructive rebuild. modernc.org/sqlite's bundled SQLite
	// doesn't support ADD COLUMN IF NOT EXISTS, so check first.
	if err := addColumnIfMissing(db, "peer_allocations", "user_id", "TEXT"); err != nil {
		return nil, err
	}
	if err := addColumnIfMissing(db, "peer_allocations", "location_id", "TEXT"); err != nil {
		return nil, err
	}

	return &PeerStore{db: db, subnetBase: subnetBase}, nil
}

func addColumnIfMissing(db *sql.DB, table, column, sqlType string) error {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return fmt.Errorf("inspecting %s schema: %w", table, err)
	}
	defer rows.Close()

	var name string
	var cid, notnull, pk int
	var colType, dflt sql.NullString
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &colType, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("reading %s schema: %w", table, err)
		}
		if name == column {
			return nil // already present
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, sqlType)); err != nil {
		return fmt.Errorf("adding %s column: %w", column, err)
	}
	return nil
}

func (p *PeerStore) Close() error {
	return p.db.Close()
}

// AllocateIP assigns (or returns the existing) tunnel-internal IP for
// publicKey, out of p.subnetBase.0.0/16. .0 and .1 in each /24 are skipped
// (network address and a reserved slot for the server itself). userID and
// locationID attribute the peer for usage reporting (see StatsReporter) —
// every /connect call generates a brand new keypair (no key reuse across
// reconnects), so a single user can accumulate many peer rows over time;
// usage aggregation sums across all of a user's public keys, not just one.
func (p *PeerStore) AllocateIP(publicKey, userID, locationID string) (string, error) {
	var existing string
	err := p.db.QueryRow(`SELECT assigned_ip FROM peer_allocations WHERE public_key = ?`, publicKey).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("looking up existing allocation: %w", err)
	}

	res, err := p.db.Exec(
		`INSERT INTO peer_allocations (public_key, assigned_ip, user_id, location_id) VALUES (?, '', ?, ?)`,
		publicKey, userID, locationID,
	)
	if err != nil {
		return "", fmt.Errorf("inserting allocation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return "", fmt.Errorf("reading allocation id: %w", err)
	}

	ip := sequenceToIP(id, p.subnetBase)
	if _, err := p.db.Exec(`UPDATE peer_allocations SET assigned_ip = ? WHERE id = ?`, ip, id); err != nil {
		return "", fmt.Errorf("saving assigned ip: %w", err)
	}
	return ip, nil
}

// PeerInfo is one row from peer_allocations, joined with nothing else —
// callers needing the user's device_id or tier join against users.Store
// themselves (kept decoupled since PeerStore and users.Store are separate
// SQLite handles opened independently, per docs/BACKEND.md).
type PeerInfo struct {
	PublicKey  string
	AssignedIP string
	UserID     string
	LocationID string
	CreatedAt  time.Time
}

// AllPeers returns every known peer allocation — used by StatsReporter to
// correlate live wgctrl device stats (keyed by public key) back to a user
// and location for the admin usage dashboard.
func (p *PeerStore) AllPeers() ([]PeerInfo, error) {
	rows, err := p.db.Query(`SELECT public_key, assigned_ip, COALESCE(user_id, ''), COALESCE(location_id, ''), created_at FROM peer_allocations`)
	if err != nil {
		return nil, fmt.Errorf("querying peer allocations: %w", err)
	}
	defer rows.Close()

	var out []PeerInfo
	for rows.Next() {
		var pi PeerInfo
		if err := rows.Scan(&pi.PublicKey, &pi.AssignedIP, &pi.UserID, &pi.LocationID, &pi.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning peer allocation: %w", err)
		}
		out = append(out, pi)
	}
	return out, rows.Err()
}

func sequenceToIP(seq int64, subnetBase string) string {
	offset := seq + 1 // start at .2, not .0/.1
	octet3 := (offset / 254) % 256
	octet4 := (offset % 254) + 2
	return fmt.Sprintf("%s.%d.%d", subnetBase, octet3, octet4)
}
