// amnezia-setup is a one-shot CLI that configures a running AmneziaWG
// userspace interface (private key, listen port, obfuscation params) via
// awgctrl-go's UAPI client — the same library backend/ itself uses to
// register peers — rather than requiring amneziawg-tools (awg/awg-quick) to
// be installed on the host. Prints the resulting public key on success,
// matching what infra/scripts/setup-central-server.sh prints for the
// kernel-WireGuard path. See infra/scripts/setup-amnezia-interface.sh for
// how this fits into bringing up the interface end to end, and
// backend/internal/servers/amnezia.go for what the AWG_* obfuscation
// params mean.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"github.com/advanced-wg/awgctrl-go"
	"github.com/advanced-wg/awgctrl-go/wgtypes"
	"golang.org/x/crypto/curve25519"
)

func genKeyPair() (priv, pub [32]byte, err error) {
	if _, err = rand.Read(priv[:]); err != nil {
		return
	}
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	pubSlice, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return
	}
	copy(pub[:], pubSlice)
	return
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: amnezia-setup <interface> <listen-port> <key-dir>")
		os.Exit(1)
	}
	iface := os.Args[1]
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad port:", err)
		os.Exit(1)
	}
	keyDir := os.Args[3]

	privKeyPath := keyDir + "/" + iface + "_private.key"
	pubKeyPath := keyDir + "/" + iface + "_public.key"

	// Reuses an existing key if this has run before (e.g. re-applying AWG_*
	// param changes after a redeploy) rather than rotating the server's
	// identity every time, which would strand every previously-issued
	// client config.
	var privKey wgtypes.Key
	if b, err := os.ReadFile(privKeyPath); err == nil {
		privKey, err = wgtypes.ParseKey(string(b))
		if err != nil {
			fmt.Fprintln(os.Stderr, "parsing existing private key:", err)
			os.Exit(1)
		}
	} else {
		privRaw, pubRaw, err := genKeyPair()
		if err != nil {
			fmt.Fprintln(os.Stderr, "generating keypair:", err)
			os.Exit(1)
		}
		privKey, _ = wgtypes.NewKey(privRaw[:])
		if err := os.WriteFile(privKeyPath, []byte(base64.StdEncoding.EncodeToString(privRaw[:])), 0600); err != nil {
			fmt.Fprintln(os.Stderr, "writing private key:", err)
			os.Exit(1)
		}
		_ = os.WriteFile(pubKeyPath, []byte(base64.StdEncoding.EncodeToString(pubRaw[:])), 0644)
	}

	client, err := wgctrl.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "connecting to WireGuard control plane:", err)
		os.Exit(1)
	}
	defer client.Close()

	cfg := wgtypes.Config{
		PrivateKey: &privKey,
		ListenPort: &port,
	}

	// Amnezia obfuscation params, read from AWG_* env vars — same values
	// backend/internal/servers/amnezia.go expects, so client configs built
	// by the real backend against this interface will actually match.
	get := func(name string) int {
		v, _ := strconv.Atoi(os.Getenv(name))
		return v
	}
	if os.Getenv("AWG_JC") != "" {
		jc, jmin, jmax := get("AWG_JC"), get("AWG_JMIN"), get("AWG_JMAX")
		s1, s2, s3, s4 := get("AWG_S1"), get("AWG_S2"), get("AWG_S3"), get("AWG_S4")
		h1, h2, h3, h4 := os.Getenv("AWG_H1"), os.Getenv("AWG_H2"), os.Getenv("AWG_H3"), os.Getenv("AWG_H4")
		i1, i2, i3, i4, i5 := os.Getenv("AWG_I1"), os.Getenv("AWG_I2"), os.Getenv("AWG_I3"), os.Getenv("AWG_I4"), os.Getenv("AWG_I5")
		cfg.Jc, cfg.Jmin, cfg.Jmax = &jc, &jmin, &jmax
		cfg.S1, cfg.S2, cfg.S3, cfg.S4 = &s1, &s2, &s3, &s4
		cfg.H1, cfg.H2, cfg.H3, cfg.H4 = &h1, &h2, &h3, &h4
		cfg.I1, cfg.I2, cfg.I3, cfg.I4, cfg.I5 = &i1, &i2, &i3, &i4, &i5
	}

	if err := client.ConfigureDevice(context.Background(), iface, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "configuring device:", err)
		os.Exit(1)
	}

	fmt.Println(privKey.PublicKey().String())
}
