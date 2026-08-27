package servers

import (
	"context"
	"fmt"
	"net"

	"github.com/advanced-wg/awgctrl-go"
	"github.com/advanced-wg/awgctrl-go/wgtypes"
)

// RegisterPeer adds/updates a peer on the live WireGuard interface named
// interfaceName (e.g. "wg0" or "awg0", created by
// infra/scripts/setup-central-server.sh) so it will actually accept
// connections from this client — this is the step that was previously
// missing: handleConnect generated keys and a config but never told any
// real WireGuard interface about them.
//
// amnezia must match whatever the interface itself was configured with (see
// ConfigureAmneziaDevice) — it sets AdvancedSecurity on the peer, which is
// what actually turns on AmneziaWG obfuscation for its traffic; a peer
// added without it on an AmneziaWG-configured device would still work but
// wouldn't be obfuscated.
//
// Uses awgctrl-go (a wgctrl fork that also understands AmneziaWG device/peer
// fields; talks to the interface via netlink or its userspace UAPI socket)
// rather than shelling out to the `wg`/`awg` CLI — no parsing command
// output, and it fails cleanly (interface not found) when run somewhere
// without a real WireGuard interface, which is expected everywhere except
// the actual central server.
func RegisterPeer(interfaceName, clientPublicKeyBase64, allowedIP string, amnezia bool) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("connecting to WireGuard control plane: %w", err)
	}
	defer client.Close()

	pubKey, err := wgtypes.ParseKey(clientPublicKeyBase64)
	if err != nil {
		return fmt.Errorf("parsing client public key: %w", err)
	}

	_, allowedNet, err := net.ParseCIDR(allowedIP + "/32")
	if err != nil {
		return fmt.Errorf("parsing allowed ip %q: %w", allowedIP, err)
	}

	return client.ConfigureDevice(context.Background(), interfaceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:        pubKey,
				AllowedIPs:       []net.IPNet{*allowedNet},
				AdvancedSecurity: amnezia,
			},
		},
	})
}

// RemovePeer drops a peer from the live WireGuard interface so it can no
// longer connect or reconnect — the server rejects its handshake outright
// from this point on, same as it would for a public key it never knew
// about. Does not touch peer_allocations (see PeerStore: allocations are an
// append-only historical record, not reclaimed), so AllPeers/stats keep the
// row but it simply stops reporting connected once the interface drops it.
func RemovePeer(interfaceName, clientPublicKeyBase64 string) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("connecting to WireGuard control plane: %w", err)
	}
	defer client.Close()

	pubKey, err := wgtypes.ParseKey(clientPublicKeyBase64)
	if err != nil {
		return fmt.Errorf("parsing client public key: %w", err)
	}

	return client.ConfigureDevice(context.Background(), interfaceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: pubKey,
				Remove:    true,
			},
		},
	})
}

// ConfigureAmneziaDevice applies device-level AmneziaWG obfuscation
// parameters to interfaceName. Called once at startup (see cmd/server/main.go)
// rather than per-peer: Jc/Jmin/.../I5 are shared by the whole interface, so
// every client must be handed configs built (see BuildClientConfig) with the
// same params this call applies — that's why they come from AWG_* env vars
// rather than being generated fresh here.
//
// Idempotent — safe to call on every backend startup even if the interface
// was already configured (e.g. by a prior backend instance), since it just
// reasserts the same values.
func ConfigureAmneziaDevice(interfaceName string, params AmneziaParams) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("connecting to WireGuard control plane: %w", err)
	}
	defer client.Close()

	return client.ConfigureDevice(context.Background(), interfaceName, params.wgtypesConfig())
}
