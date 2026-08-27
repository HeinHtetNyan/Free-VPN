package servers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ResyncFriends re-registers every still-active "friend" peer (issued via
// POST /admin/friends, see handleAdminCreateFriend) onto interfaceName —
// called once at startup. Peers aren't part of the WireGuard interface's
// persistent state: a host reboot recreates the raw interface with no peers
// at all, and unlike app users (whose next /connect call naturally
// re-registers them via RegisterPeer), a friend's client never calls any
// endpoint of its own, so nothing else would ever bring them back.
//
// Revocation status only lives on the Activation-Licenses side (see that
// project's VpnFriend model) — peer_allocations here is an append-only
// local record that never reflects revocation (see PeerStore's own doc
// comment), so this asks the admin backend which friends are still active
// rather than trusting local state to decide who should regain access.
//
// Best-effort: logs and returns nil on any failure rather than blocking
// startup, same as ConfigureAmneziaDevice's caller in cmd/server/main.go.
// A no-op with ingestURL/token unset, matching reporter.Run's own pattern —
// reuses the same ADMIN_INGEST_URL/ADMIN_INGEST_TOKEN env vars.
func ResyncFriends(peers *PeerStore, interfaceName, ingestURL, token string, amnezia bool) error {
	if ingestURL == "" || token == "" {
		return nil
	}

	activeURL := strings.TrimSuffix(ingestURL, "/ingest") + "/friends/active"
	req, err := http.NewRequest(http.MethodGet, activeURL, nil)
	if err != nil {
		return fmt.Errorf("building friends/active request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching active friends: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching active friends: unexpected status %d", resp.StatusCode)
	}

	var body struct {
		PublicKeys []string `json:"public_keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decoding active friends: %w", err)
	}

	allPeers, err := peers.AllPeers()
	if err != nil {
		return fmt.Errorf("loading local peer allocations: %w", err)
	}
	ipByKey := make(map[string]string, len(allPeers))
	for _, p := range allPeers {
		ipByKey[p.PublicKey] = p.AssignedIP
	}

	for _, pubKey := range body.PublicKeys {
		ip, ok := ipByKey[pubKey]
		if !ok {
			log.Printf("friend resync: no local IP allocation for %s, skipping", pubKey)
			continue
		}
		if err := RegisterPeer(interfaceName, pubKey, ip, amnezia); err != nil {
			log.Printf("friend resync: could not register %s: %v", pubKey, err)
			continue
		}
		log.Printf("friend resync: re-registered %s -> %s", pubKey, ip)
	}
	return nil
}
