package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"sy-vpn-backend/internal/api"
	"sy-vpn-backend/internal/reporter"
	"sy-vpn-backend/internal/reports"
	"sy-vpn-backend/internal/servers"
	"sy-vpn-backend/internal/stats"
	"sy-vpn-backend/internal/users"
)

func main() {
	locations, err := servers.LoadLocations()
	if err != nil {
		log.Fatalf("loading locations: %v", err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "vpnapp.db"
	}
	userStore, err := users.NewStore(dbPath)
	if err != nil {
		log.Fatalf("opening user store at %s: %v", dbPath, err)
	}
	defer userStore.Close()

	peerStore, err := servers.NewPeerStore(dbPath)
	if err != nil {
		log.Fatalf("opening peer store at %s: %v", dbPath, err)
	}
	defer peerStore.Close()

	reportStore, err := reports.NewStore(dbPath)
	if err != nil {
		log.Fatalf("opening report store at %s: %v", dbPath, err)
	}
	defer reportStore.Close()

	wgIface := os.Getenv("WG_INTERFACE")
	if wgIface == "" {
		wgIface = "wg0"
	}

	serverPublicKey := os.Getenv("SERVER_PUBLIC_KEY")
	if serverPublicKey == "" {
		serverPublicKey = api.DefaultServerPublicKeyPlaceholder
	}

	// AmneziaWG obfuscation — see internal/servers/amnezia.go and
	// docs/ARCHITECTURE.md "Censorship resistance". Absent AWG_* env vars
	// (infra/local-test, or before infra/scripts/generate-amnezia-params.sh
	// has been run against a real deployment), this is plain WireGuard,
	// same as every deployment before this feature existed.
	amneziaParams, amneziaEnabled, err := servers.AmneziaParamsFromEnv()
	if err != nil {
		log.Fatalf("AmneziaWG config: %v", err)
	}
	if amneziaEnabled {
		if err := servers.ConfigureAmneziaDevice(wgIface, amneziaParams); err != nil {
			log.Printf("warning: could not apply AmneziaWG params to interface %q (expected until a real AmneziaWG-capable interface is deployed): %v", wgIface, err)
		} else {
			log.Printf("AmneziaWG obfuscation active on %q", wgIface)
		}
	}

	// Polls wgctrl for live peer stats (connected-now count, per-user
	// traffic) — see internal/stats. 30s balances freshness against load on
	// a single small VPS; the admin push below reuses this same collector
	// rather than polling wgctrl a second time.
	statsCollector := stats.NewCollector(wgIface, peerStore)
	go statsCollector.Run(30*time.Second, nil)

	// Optional: pushes usage snapshots and undelivered issue reports to a
	// separate admin dashboard project (Activation-Licenses) — no-ops if
	// unset. See internal/reporter.
	go reporter.Run(
		statsCollector,
		reportStore,
		os.Getenv("ADMIN_INGEST_URL"),
		os.Getenv("ADMIN_INGEST_TOKEN"),
		60*time.Second,
		nil,
	)

	server := api.NewServer(userStore, peerStore, locations, wgIface, serverPublicKey, amneziaParams, amneziaEnabled, statsCollector, reportStore)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Defaults to all interfaces (0.0.0.0) since infra/local-test's client
	// container reaches the server container by IP over a Docker bridge
	// network. Production (running with network_mode: host, so it can reach
	// the host's real wg0 via netlink) sets this to 127.0.0.1 explicitly —
	// under host networking there's no Docker port-publish step to fall back
	// on for keeping this off the public interface, unlike the other apps on
	// that VPS.
	bindHost := os.Getenv("BIND_HOST")
	addr := bindHost + ":" + port
	log.Printf("control plane listening on %s (%d locations loaded)", addr, len(locations))
	if err := http.ListenAndServe(addr, server.Router()); err != nil {
		log.Fatal(err)
	}
}
