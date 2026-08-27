// AmneziaWG obfuscation support. Plain WireGuard's handshake has a
// recognizable packet-size/timing signature that DPI (deployed by, among
// others, Myanmar ISPs — see docs/ARCHITECTURE.md "Censorship resistance")
// can fingerprint in roughly 100 packets, independent of port or destination
// IP. AmneziaWG defeats this by sending junk packets and padding real ones
// before the handshake, using the wgctrl fork github.com/advanced-wg/awgctrl-go
// (the stock golang.zx2c4.com/wireguard/wgctrl has no concept of this).
//
// These parameters are DEVICE-level (one shared value per WireGuard
// interface, applied once at startup — see ConfigureAmneziaDevice), not
// per-peer: every client's [Interface] block must carry the exact same
// values or its handshake will not parse. That's why they're generated once
// (see infra/scripts/generate-amnezia-params.sh, run by hand, output pasted
// into .env) rather than regenerated on every server restart or /connect
// call, which would strand every client whose config was built with the
// previous values.
package servers

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/advanced-wg/awgctrl-go/wgtypes"
)

// AmneziaParams is one complete, matched set of AmneziaWG obfuscation
// values — see docs/ARCHITECTURE.md and the awgctrl-go wgtypes.Config docs
// for what each field means. The zero value is "not configured": callers
// check AmneziaEnabled (derived from whether env vars were set) rather than
// inspecting these fields directly.
type AmneziaParams struct {
	Jc, Jmin, Jmax     int
	S1, S2, S3, S4     int
	H1, H2, H3, H4     string
	I1, I2, I3, I4, I5 string
}

// amneziaEnvVars lists every AWG_* env var AmneziaParamsFromEnv requires.
// Keep in sync with the struct above and docs/OPEN_QUESTIONS.md.
var amneziaEnvVars = []string{
	"AWG_JC", "AWG_JMIN", "AWG_JMAX",
	"AWG_S1", "AWG_S2", "AWG_S3", "AWG_S4",
	"AWG_H1", "AWG_H2", "AWG_H3", "AWG_H4",
	"AWG_I1", "AWG_I2", "AWG_I3", "AWG_I4", "AWG_I5",
}

// AmneziaParamsFromEnv reads AWG_* env vars. ok is false if none are set
// (plain WireGuard — e.g. infra/local-test, or before generate-amnezia-params.sh
// has been run against a real deployment) or an error if some but not all
// are set (a half-configured deployment is worse than none: it would hand
// out client configs the server-side device doesn't actually match).
func AmneziaParamsFromEnv() (params AmneziaParams, ok bool, err error) {
	present := 0
	for _, name := range amneziaEnvVars {
		if os.Getenv(name) != "" {
			present++
		}
	}
	if present == 0 {
		return AmneziaParams{}, false, nil
	}
	if present != len(amneziaEnvVars) {
		return AmneziaParams{}, false, fmt.Errorf("partial AmneziaWG config: %d/%d AWG_* env vars set (need all or none)", present, len(amneziaEnvVars))
	}

	atoi := func(name string) (int, error) {
		v, err := strconv.Atoi(os.Getenv(name))
		if err != nil {
			return 0, fmt.Errorf("parsing %s: %w", name, err)
		}
		return v, nil
	}

	var e error
	get := func(name string) int {
		v, err := atoi(name)
		if err != nil && e == nil {
			e = err
		}
		return v
	}

	p := AmneziaParams{
		Jc: get("AWG_JC"), Jmin: get("AWG_JMIN"), Jmax: get("AWG_JMAX"),
		S1: get("AWG_S1"), S2: get("AWG_S2"), S3: get("AWG_S3"), S4: get("AWG_S4"),
		H1: os.Getenv("AWG_H1"), H2: os.Getenv("AWG_H2"), H3: os.Getenv("AWG_H3"), H4: os.Getenv("AWG_H4"),
		I1: os.Getenv("AWG_I1"), I2: os.Getenv("AWG_I2"), I3: os.Getenv("AWG_I3"), I4: os.Getenv("AWG_I4"), I5: os.Getenv("AWG_I5"),
	}
	if e != nil {
		return AmneziaParams{}, false, e
	}
	return p, true, nil
}

// wgtypesConfig converts to the pointer-based fields wgtypes.Config expects
// for ConfigureDevice — see ConfigureAmneziaDevice.
func (p AmneziaParams) wgtypesConfig() wgtypes.Config {
	i := func(v int) *int { return &v }
	s := func(v string) *string { return &v }
	return wgtypes.Config{
		Jc: i(p.Jc), Jmin: i(p.Jmin), Jmax: i(p.Jmax),
		S1: i(p.S1), S2: i(p.S2), S3: i(p.S3), S4: i(p.S4),
		H1: s(p.H1), H2: s(p.H2), H3: s(p.H3), H4: s(p.H4),
		I1: s(p.I1), I2: s(p.I2), I3: s(p.I3), I4: s(p.I4), I5: s(p.I5),
	}
}

// interfaceLines renders the AmneziaWG key-value lines BuildClientConfig
// appends to a client's [Interface] block. Every connecting client must
// carry the exact values the server interface was configured with (see
// ConfigureAmneziaDevice) or its handshake won't parse.
func (p AmneziaParams) interfaceLines() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Jc = %d\nJmin = %d\nJmax = %d\n", p.Jc, p.Jmin, p.Jmax)
	fmt.Fprintf(&b, "S1 = %d\nS2 = %d\nS3 = %d\nS4 = %d\n", p.S1, p.S2, p.S3, p.S4)
	fmt.Fprintf(&b, "H1 = %s\nH2 = %s\nH3 = %s\nH4 = %s\n", p.H1, p.H2, p.H3, p.H4)
	fmt.Fprintf(&b, "I1 = %s\nI2 = %s\nI3 = %s\nI4 = %s\nI5 = %s\n", p.I1, p.I2, p.I3, p.I4, p.I5)
	return b.String()
}
