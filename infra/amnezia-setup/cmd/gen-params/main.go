// gen-params generates one matched set of AmneziaWG obfuscation parameters
// and prints them as AWG_* env var lines — see
// infra/scripts/generate-amnezia-params.sh and
// backend/internal/servers/amnezia.go for why this is a one-time, run-by-hand
// step rather than something the server generates itself at boot: every
// connecting client's config must carry the exact same values the server
// interface was configured with, so regenerating them on every restart
// would strand already-issued client configs.
package main

import (
	"fmt"

	"github.com/advanced-wg/awgctrl-go/wgtypes"
)

func main() {
	var cfg wgtypes.Config
	cfg.GenerateAmneziaParams()
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	fmt.Printf("AWG_JC=%d\n", *cfg.Jc)
	fmt.Printf("AWG_JMIN=%d\n", *cfg.Jmin)
	fmt.Printf("AWG_JMAX=%d\n", *cfg.Jmax)
	fmt.Printf("AWG_S1=%d\n", *cfg.S1)
	fmt.Printf("AWG_S2=%d\n", *cfg.S2)
	fmt.Printf("AWG_S3=%d\n", *cfg.S3)
	fmt.Printf("AWG_S4=%d\n", *cfg.S4)
	fmt.Printf("AWG_H1=%s\n", *cfg.H1)
	fmt.Printf("AWG_H2=%s\n", *cfg.H2)
	fmt.Printf("AWG_H3=%s\n", *cfg.H3)
	fmt.Printf("AWG_H4=%s\n", *cfg.H4)
	fmt.Printf("AWG_I1=%s\n", *cfg.I1)
	fmt.Printf("AWG_I2=%s\n", *cfg.I2)
	fmt.Printf("AWG_I3=%s\n", *cfg.I3)
	fmt.Printf("AWG_I4=%s\n", *cfg.I4)
	fmt.Printf("AWG_I5=%s\n", *cfg.I5)
}
