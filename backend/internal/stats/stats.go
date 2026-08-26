// Package stats periodically reads live WireGuard peer data (via wgctrl,
// the same control plane internal/servers/wireguard_manager.go registers
// peers on) and correlates it with peer_allocations to answer two things:
// "how many devices are connected right now" (api.handleStats, public,
// aggregate-only) and per-user/per-location usage detail (pushed to the
// admin dashboard in a separate project — see docs/DECISIONS.md).
package stats

import (
	"fmt"
	"log"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"

	"sy-vpn-backend/internal/servers"
)

// OnlineWindow: a peer counts as "connected" if its last WireGuard handshake
// was within this long ago. WireGuard's own PersistentKeepalive (25s, see
// servers.BuildClientConfig) forces a rekey roughly every couple of minutes
// on a genuinely active tunnel, so 3 minutes gives headroom for one missed
// cycle without flip-flopping a real connection to "offline".
const OnlineWindow = 3 * time.Minute

// PeerSnapshot is one peer_allocations row as of the most recent poll,
// enriched with live wgctrl data where the peer is currently known to wg0.
type PeerSnapshot struct {
	PublicKey     string
	UserID        string
	LocationID    string
	AssignedIP    string
	Connected     bool
	LastHandshake time.Time
	RxBytesTotal  int64
	TxBytesTotal  int64
	// Instantaneous throughput (bytes/sec) computed over this poll's
	// interval — not derived from a cumulative counter by the reader, so no
	// restart-of-counter edge cases downstream (mirrors the pattern used by
	// Activation-Licenses' VPS monitoring agent).
	RxBps float64
	TxBps float64
}

type Snapshot struct {
	TakenAt      time.Time
	ConnectedNow int
	TotalDevices int
	Peers        []PeerSnapshot
}

type byteSample struct {
	at time.Time
	rx int64
	tx int64
}

// Collector polls wgctrl on an interval and keeps the latest Snapshot in
// memory — both api.handleStats (public, aggregate-only) and any admin
// usage reporter read from Current() rather than each polling wgctrl
// independently.
type Collector struct {
	wgIface string
	peers   *servers.PeerStore

	mu        sync.RWMutex
	current   Snapshot
	lastBytes map[string]byteSample
}

func NewCollector(wgIface string, peerStore *servers.PeerStore) *Collector {
	return &Collector{wgIface: wgIface, peers: peerStore, lastBytes: map[string]byteSample{}}
}

// Current returns the most recent snapshot. Zero-value (empty) until the
// first Poll completes.
func (c *Collector) Current() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}

// Poll runs one collection cycle. Safe to call directly (e.g. for tests);
// Run wraps this in a ticker loop for production use.
func (c *Collector) Poll() error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("connecting to WireGuard control plane: %w", err)
	}
	defer client.Close()

	dev, err := client.Device(c.wgIface)
	if err != nil {
		return fmt.Errorf("reading device %q (expected until a central server is deployed): %w", c.wgIface, err)
	}

	liveByKey := make(map[string]int, len(dev.Peers))
	for i, p := range dev.Peers {
		liveByKey[p.PublicKey.String()] = i
	}

	allocs, err := c.peers.AllPeers()
	if err != nil {
		return fmt.Errorf("listing peer allocations: %w", err)
	}

	now := time.Now()
	c.mu.RLock()
	prevBytes := c.lastBytes
	c.mu.RUnlock()

	peerSnaps := make([]PeerSnapshot, 0, len(allocs))
	newBytes := make(map[string]byteSample, len(allocs))
	connectedNow := 0

	for _, a := range allocs {
		snap := PeerSnapshot{
			PublicKey:  a.PublicKey,
			UserID:     a.UserID,
			LocationID: a.LocationID,
			AssignedIP: a.AssignedIP,
		}
		if idx, ok := liveByKey[a.PublicKey]; ok {
			live := dev.Peers[idx]
			snap.LastHandshake = live.LastHandshakeTime
			snap.RxBytesTotal = live.ReceiveBytes
			snap.TxBytesTotal = live.TransmitBytes
			snap.Connected = !live.LastHandshakeTime.IsZero() && now.Sub(live.LastHandshakeTime) <= OnlineWindow

			if prev, had := prevBytes[a.PublicKey]; had {
				if dt := now.Sub(prev.at).Seconds(); dt > 0 {
					if live.ReceiveBytes >= prev.rx {
						snap.RxBps = float64(live.ReceiveBytes-prev.rx) / dt
					}
					if live.TransmitBytes >= prev.tx {
						snap.TxBps = float64(live.TransmitBytes-prev.tx) / dt
					}
				}
			}
			newBytes[a.PublicKey] = byteSample{at: now, rx: live.ReceiveBytes, tx: live.TransmitBytes}
		}
		if snap.Connected {
			connectedNow++
		}
		peerSnaps = append(peerSnaps, snap)
	}

	c.mu.Lock()
	c.current = Snapshot{TakenAt: now, ConnectedNow: connectedNow, TotalDevices: len(allocs), Peers: peerSnaps}
	c.lastBytes = newBytes
	c.mu.Unlock()
	return nil
}

// Run polls on the given interval until stop is closed. Meant to be started
// in its own goroutine from cmd/server/main.go.
func (c *Collector) Run(interval time.Duration, stop <-chan struct{}) {
	if err := c.Poll(); err != nil {
		log.Printf("stats: initial poll: %v", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.Poll(); err != nil {
				log.Printf("stats: poll: %v", err)
			}
		case <-stop:
			return
		}
	}
}
