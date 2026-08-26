# Backend (control plane)

Language: **Go**. Chosen because it pairs natively with WireGuard tooling (`wgctrl-go`, and WireGuard's own userspace implementation is Go), and because the same binary/process style suits a small always-on VPS well.

## Responsibilities

- **Auth** — issuing/validating whatever identity the app uses (see `docs/OPEN_QUESTIONS.md`, not yet decided: real accounts vs. anonymous device ID).
- **User/tier state** — free-with-ads by default; a flag reserved for a future paid tier (not yet designed).
- **Location list** — the set of locations the app can show, and which LocalToNet relay address + which central WireGuard server backs each one.
- **WireGuard peer provisioning** — generating a keypair for a device (or reusing one), registering the public key on the central WireGuard server, returning a ready-to-use config scoped to the requested location's relay address.

The control plane itself does **not** carry VPN traffic — see `docs/ARCHITECTURE.md`.

## Planned structure

```
backend/
├── go.mod
├── cmd/
│   └── server/        entrypoint (main.go) — wires everything together, starts HTTP server
└── internal/
    ├── api/            HTTP handlers / routing
    ├── auth/            identity + session/token logic
    ├── users/           user + tier records
    └── servers/         location list + WireGuard peer provisioning
```

`internal/` is intentionally not importable outside this module — Go convention for "this is app-internal, not a public library."

## Current state (implemented 2026-08-26)

Working end-to-end skeleton, built and tested with `go build`/manual curl requests:

- `POST /auth/register {device_id}` — anonymous, idempotent registration; returns a bearer token. Same `device_id` always resolves to the same account (`internal/users.Store.GetOrCreateByDeviceID`).
- `GET /locations` (auth required) — returns the picker list from `internal/servers/locations.json`, without exposing relay/central-server internals to the client.
- `POST /connect {location_id}` (auth required) — generates a real WireGuard Curve25519 keypair (`internal/servers/wireguard.go`, using `golang.org/x/crypto/curve25519`, mirroring `wg genkey | wg pubkey`) and returns a client `.conf`. **The returned config won't actually connect to anything yet** — `Endpoint` and the server's public key are placeholders until a real LocalToNet tunnel + central WireGuard server exist (see `docs/OPEN_QUESTIONS.md`).
- `GET /health`.

Not yet done: registering the generated client public key as an actual peer on a live WireGuard server (there isn't one deployed yet — that's an infra step, not a code gap).

## Decisions now resolved (see docs/DECISIONS.md)

- **Auth strategy: anonymous device-bound identity**, not real accounts. Chosen for frictionless onboarding (matches the "really easy to use" priority) and because a censorship-circumvention tool arguably shouldn't ask for an email up front. Tradeoff accepted: no cross-device subscription restore without adding real accounts later — acceptable since paid tier isn't designed yet anyway.
- **WireGuard peer management: direct, in-process.** Control plane and WireGuard now run on the same central server, so provisioning can shell out to `wg`/`wg-quick` directly (or use `wgctrl-go`) once a real server exists — no remote-management protocol needed.

## Persistence and WireGuard integration (implemented 2026-08-26)

- **Database: SQLite** (`modernc.org/sqlite` — pure Go, no CGO), one file (`DB_PATH` env var, default `vpnapp.db`) holding both the `users` table and a `peer_allocations` table (`internal/servers/peer_store.go`) that assigns each WireGuard client a stable tunnel IP out of `10.66.0.0/16`. Verified persisting across a server restart (registered a device, killed the process, restarted, same token still resolved).
- **Live peer registration**: `handleConnect` now calls `servers.RegisterPeer` (`internal/servers/wireguard_manager.go`, using `wgctrl` — talks to the kernel WireGuard interface via netlink, no shelling out to the `wg` CLI) to actually add the client as a peer on interface `WG_INTERFACE` (env var, default `wg0`). This is **best-effort**: since no real central server exists yet, it fails cleanly and just logs a warning ("operation not permitted" / interface not found) rather than breaking the request — verified this exact behavior locally. **Once a real central server is deployed, promote this to a hard failure** (`api/handlers.go` `handleConnect`, marked with a TODO comment) — a config that was never actually registered as a peer will never connect, and silently handing one out at that point would just be confusing rather than a reasonable fallback.

## Mechanism validated end-to-end (2026-08-26)

`infra/local-test/` proves this code — real key generation, real `wgctrl` peer registration, a real WireGuard handshake, real NAT'd internet egress — actually works, using two disposable Docker containers and touching no host networking. See `docs/DECISIONS.md` for what exactly was proven and the (Docker-specific, not production-relevant) friction hit along the way.

Also: the central server's public key is now configurable via `SERVER_PUBLIC_KEY` (env var, defaults to `api.DefaultServerPublicKeyPlaceholder`), matching `WG_INTERFACE`/`DB_PATH` — no code edit needed once a real server exists, just set the three env vars.

## Still open

- **Live central server.** The above is real, tested, *validated* code — it just has nothing to register peers *against* in production yet. See `infra/scripts/setup-central-server.sh` (a complete WireGuard+NAT+systemd setup, not just a package install) and `infra/LOCALTONET_SETUP.md`.
- **IP allocation is a simple incrementing sequence**, doesn't reclaim IPs from removed peers. Fine at current scale (~65k addresses available); revisit if that ever becomes a real constraint.
