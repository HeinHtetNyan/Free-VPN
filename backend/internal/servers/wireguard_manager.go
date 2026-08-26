package servers

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// RegisterPeer adds/updates a peer on the live WireGuard interface named
// interfaceName (e.g. "wg0", created by infra/scripts/setup-central-server.sh)
// so it will actually accept connections from this client — this is the
// step that was previously missing: handleConnect generated keys and a
// config but never told any real WireGuard interface about them.
//
// Uses wgctrl (talks to the kernel WireGuard interface via netlink) rather
// than shelling out to the `wg` CLI — no parsing command output, and it
// fails cleanly (interface not found) when run somewhere without a real
// WireGuard interface, which is expected everywhere except the actual
// central server.
func RegisterPeer(interfaceName, clientPublicKeyBase64, allowedIP string) error {
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

	return client.ConfigureDevice(interfaceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  pubKey,
				AllowedIPs: []net.IPNet{*allowedNet},
			},
		},
	})
}
