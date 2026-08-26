// Package reporter periodically pushes usage snapshots (internal/stats) to
// an external admin dashboard — deliberately a separate project
// (Activation-Licenses), not built into this backend, so this is a small
// HTTP push client, not a UI. Entirely optional: with no ingest URL/token
// configured (the common case for local dev and infra/local-test), Run
// simply does nothing.
package reporter

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"sy-vpn-backend/internal/stats"
)

type peerPayload struct {
	UserID          string  `json:"user_id"`
	PublicKey       string  `json:"public_key"`
	LocationID      string  `json:"location_id"`
	AssignedIP      string  `json:"assigned_ip"`
	Connected       bool    `json:"connected"`
	LastHandshakeAt *string `json:"last_handshake_at"`
	RxBytesTotal    int64   `json:"rx_bytes_total"`
	TxBytesTotal    int64   `json:"tx_bytes_total"`
	RxBps           float64 `json:"rx_bps"`
	TxBps           float64 `json:"tx_bps"`
}

type snapshotPayload struct {
	ConnectedNow int           `json:"connected_now"`
	TotalDevices int           `json:"total_devices"`
	Peers        []peerPayload `json:"peers"`
}

// Run POSTs the collector's latest snapshot to ingestURL on the given
// interval until stop is closed. No-op (logs once, returns) if ingestURL or
// token is empty — call unconditionally from main.go and let this decide
// whether there's anything to do.
func Run(collector *stats.Collector, ingestURL, token string, interval time.Duration, stop <-chan struct{}) {
	if ingestURL == "" || token == "" {
		log.Printf("reporter: ADMIN_INGEST_URL/ADMIN_INGEST_TOKEN not set, usage reporting disabled")
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	doPush := func() {
		if err := push(client, collector, ingestURL, token); err != nil {
			log.Printf("reporter: push failed: %v", err)
		}
	}

	doPush()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			doPush()
		case <-stop:
			return
		}
	}
}

func push(client *http.Client, collector *stats.Collector, ingestURL, token string) error {
	snap := collector.Current()
	peers := make([]peerPayload, 0, len(snap.Peers))
	for _, p := range snap.Peers {
		var handshake *string
		if !p.LastHandshake.IsZero() {
			s := p.LastHandshake.UTC().Format(time.RFC3339)
			handshake = &s
		}
		peers = append(peers, peerPayload{
			UserID: p.UserID, PublicKey: p.PublicKey, LocationID: p.LocationID, AssignedIP: p.AssignedIP,
			Connected: p.Connected, LastHandshakeAt: handshake,
			RxBytesTotal: p.RxBytesTotal, TxBytesTotal: p.TxBytesTotal, RxBps: p.RxBps, TxBps: p.TxBps,
		})
	}

	body, err := json.Marshal(snapshotPayload{ConnectedNow: snap.ConnectedNow, TotalDevices: snap.TotalDevices, Peers: peers})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, ingestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return &httpStatusError{resp.StatusCode}
	}
	return nil
}

type httpStatusError struct{ code int }

func (e *httpStatusError) Error() string {
	return http.StatusText(e.code)
}
