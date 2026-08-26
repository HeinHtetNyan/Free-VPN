package servers

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	priv, err := base64.StdEncoding.DecodeString(kp.PrivateKey)
	if err != nil {
		t.Fatalf("private key not valid base64: %v", err)
	}
	if len(priv) != 32 {
		t.Fatalf("expected 32-byte private key, got %d bytes", len(priv))
	}

	pub, err := base64.StdEncoding.DecodeString(kp.PublicKey)
	if err != nil {
		t.Fatalf("public key not valid base64: %v", err)
	}
	if len(pub) != 32 {
		t.Fatalf("expected 32-byte public key, got %d bytes", len(pub))
	}

	if kp.PrivateKey == kp.PublicKey {
		t.Fatal("private and public key should differ")
	}
}

func TestGenerateKeyPair_ProducesDistinctKeys(t *testing.T) {
	a, _ := GenerateKeyPair()
	b, _ := GenerateKeyPair()
	if a.PrivateKey == b.PrivateKey {
		t.Fatal("two calls produced the same private key")
	}
}

func TestBuildClientConfig(t *testing.T) {
	loc := Location{ID: "singapore", RelayAddress: "relay.example.com:51820"}
	keys := KeyPair{PrivateKey: "CLIENT_PRIVATE_KEY", PublicKey: "CLIENT_PUBLIC_KEY"}

	config := BuildClientConfig(loc, keys, "SERVER_PUBLIC_KEY", "10.66.0.5")

	for _, want := range []string{
		"PrivateKey = CLIENT_PRIVATE_KEY",
		"Address = 10.66.0.5/32",
		"PublicKey = SERVER_PUBLIC_KEY",
		"Endpoint = relay.example.com:51820",
		"AllowedIPs = 0.0.0.0/0, ::/0",
	} {
		if !strings.Contains(config, want) {
			t.Errorf("expected config to contain %q, got:\n%s", want, config)
		}
	}
}
