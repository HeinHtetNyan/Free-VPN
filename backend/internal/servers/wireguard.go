package servers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// KeyPair is a WireGuard identity: a Curve25519 keypair, base64-encoded the
// same way `wg genkey`/`wg pubkey` produce them.
type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateKeyPair creates a new WireGuard keypair. This mirrors what
// `wg genkey | wg pubkey` does: 32 random bytes, clamped per RFC 7748,
// multiplied by the Curve25519 base point for the public half.
func GenerateKeyPair() (KeyPair, error) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return KeyPair{}, fmt.Errorf("generating private key: %w", err)
	}
	clamp(&priv)

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return KeyPair{}, fmt.Errorf("deriving public key: %w", err)
	}

	return KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(priv[:]),
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
	}, nil
}

func clamp(k *[32]byte) {
	k[0] &= 248
	k[31] &= 127
	k[31] |= 64
}

// BuildClientConfig renders a standard WireGuard client .conf for connecting
// to loc via its LocalToNet relay. serverPublicKey is the central server's
// WireGuard public key (see infra/scripts/setup-central-server.sh), and
// assignedIP is this client's address within the tunnel (see PeerStore).
// Registering clientKeys.PublicKey as an actual peer on the live server is a
// separate step — see RegisterPeer — since a config alone doesn't make a
// server accept a connection.
func BuildClientConfig(loc Location, clientKeys KeyPair, serverPublicKey, assignedIP string) string {
	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/32
DNS = 1.1.1.1

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`, clientKeys.PrivateKey, assignedIP, serverPublicKey, loc.RelayAddress)
}
