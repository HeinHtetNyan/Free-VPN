// Package servers manages the location list and WireGuard peer provisioning.
// See docs/ARCHITECTURE.md: every location's relay_address is a LocalToNet
// relay forwarding back to central_server — there is no per-location exit
// node, so exit IP is always the central server's real location.
package servers

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed locations.json
var locationsJSON []byte

type Location struct {
	ID               string `json:"id"`
	DisplayName      string `json:"display_name"`
	LocalToNetRegion string `json:"localtonet_region"`
	// RelayAddress is the LocalToNet relay endpoint (host:port) for this
	// location. Placeholder until a real LocalToNet account + tunnel exist
	// (see docs/OPEN_QUESTIONS.md).
	RelayAddress  string `json:"relay_address"`
	CentralServer string `json:"central_server"`
}

// LoadLocations reads the embedded locations.json. Kept as a simple embedded
// file rather than a database table since this list changes rarely and by
// hand (adding a location is an infra change, not a runtime event).
func LoadLocations() ([]Location, error) {
	var locs []Location
	if err := json.Unmarshal(locationsJSON, &locs); err != nil {
		return nil, fmt.Errorf("parsing locations.json: %w", err)
	}
	return locs, nil
}

// RelayHost strips the :port off RelayAddress — handed to clients so they
// can measure their own latency to the relay (see docs/MOBILE.md "Latency").
// Not new information: the full host:port is already embedded in the
// WireGuard config's Endpoint field every /connect response returns, so
// this doesn't expose anything a registered device couldn't already see.
func (l Location) RelayHost() string {
	host, _, ok := strings.Cut(l.RelayAddress, ":")
	if !ok {
		return l.RelayAddress
	}
	return host
}

func FindLocation(locs []Location, id string) (Location, bool) {
	for _, l := range locs {
		if l.ID == id {
			return l, true
		}
	}
	return Location{}, false
}
