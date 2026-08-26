package main

import (
	"log"
	"net/http"
	"os"

	"sy-vpn-backend/internal/api"
	"sy-vpn-backend/internal/servers"
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

	wgIface := os.Getenv("WG_INTERFACE")
	if wgIface == "" {
		wgIface = "wg0"
	}

	serverPublicKey := os.Getenv("SERVER_PUBLIC_KEY")
	if serverPublicKey == "" {
		serverPublicKey = api.DefaultServerPublicKeyPlaceholder
	}

	server := api.NewServer(userStore, peerStore, locations, wgIface, serverPublicKey)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("control plane listening on %s (%d locations loaded)", addr, len(locations))
	if err := http.ListenAndServe(addr, server.Router()); err != nil {
		log.Fatal(err)
	}
}
