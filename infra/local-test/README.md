# Local end-to-end test (no VPS, no LocalToNet account needed)

Proves the entire VPN mechanism works — real WireGuard key generation, live peer registration via `wgctrl`, an actual encrypted WireGuard handshake, and (by default) traffic actually exiting through the server's NAT — using two disposable, isolated Docker containers on a private Docker network. **Touches no host networking state**: no interfaces, iptables rules, or sysctls on the actual machine — see `docs/DECISIONS.md` 2026-08-26 entry for why this is safe to run anywhere Docker is installed, including a normal dev machine.

## Run it

```
./run.sh
```

This: builds the backend fresh from `../../backend`, builds two tiny Alpine images (one running the real backend + a real `wg0` interface with NAT rules, one that registers/connects through the real API and brings up the real WireGuard tunnel it gets back), runs the full flow, and tears everything down afterward.

Flags:
- `--keep` — leave the server container running afterward (for poking around with `docker exec`).
- `--scoped` — use `AllowedIPs=10.66.0.0/16` instead of the production default `0.0.0.0/0`, and skip `--privileged`. Proves the tunnel itself (handshake, encrypted transport) but not the full default-route/exit-through-server path, which needs a sysctl (`net.ipv4.conf.all.src_valid_mark`) that Docker keeps read-only for non-privileged containers even when set via `--sysctl`. Useful if you don't want to grant `--privileged` for some reason; the default (no flag) is the fuller proof and is what was actually verified when this was built.

## What a successful run looks like

- `register` / `locations` / `connect` all return real data from the real backend code.
- `wg show wg0` in the client's output shows a `latest handshake: N seconds ago` and non-zero transfer bytes — this is the real cryptographic handshake completing, not a mock.
- The ping to `10.66.0.1` (the test server's tunnel address) succeeds — real encrypted ICMP traffic through the tunnel.
- With the default (non-`--scoped`) full-tunnel mode, the final `curl` to an external URL only succeeds because the container's *entire* default route was captured by `wg0` — there is no other path out, so a successful response is real evidence the server's `iptables` MASQUERADE + `ip_forward` rules correctly relay tunnel traffic to the internet.

## Why this doesn't (and shouldn't) mirror production exactly

The real central server (`../scripts/setup-central-server.sh`) runs directly on a VPS with full root access — no Docker, no container sysctl restrictions. The friction this test harness works around (`--sysctl`, `--privileged`, `ip6tables` missing from a minimal image) is specific to proving this safely inside a container on a shared machine; a real VPS won't hit any of it. This test exists to de-risk the *design* (does the mechanism work at all) before spending money on hosting/LocalToNet, not to be a preview of how production is deployed.

## Cleanup

`run.sh` cleans up after itself (container, network, built binary) unless `--keep` is passed. The three Docker *images* (`sy-vpn-local-test-server`, `sy-vpn-local-test-client`) are left cached (harmless, ~120MB) so re-runs are fast — remove with `docker rmi sy-vpn-local-test-server sy-vpn-local-test-client` if you want them gone entirely.
