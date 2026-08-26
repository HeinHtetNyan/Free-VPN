# Infra

## What lives here

Setup for the 1-2 central servers: WireGuard server install/config, and the LocalToNet tunnels that expose it under multiple "locations." See `docs/ARCHITECTURE.md` for why this is the shape, and the tradeoffs accepted by choosing it over dedicated per-region VPS nodes.

There is **no per-region VPS provisioning** in this design — that was the alternative we didn't choose. If that changes later (see `docs/ARCHITECTURE.md` "Revisit if..."), this folder is where per-region provisioning scripts would go instead/in addition.

## Structure

```
infra/
├── scripts/
│   └── setup-central-server.sh   installs + configures WireGuard on a central VPS (complete, tested for syntax)
├── local-test/                    self-contained Docker-based proof the whole mechanism works — no VPS/LocalToNet needed
├── LOCALTONET_SETUP.md            manual checklist for the LocalToNet dashboard side (can't be scripted — needs your account)
└── locations.example.yaml         example mapping of location -> LocalToNet relay region
```

## Validated locally before spending anything on hosting

`local-test/run.sh` proves the entire mechanism — real key generation, live peer registration, an actual WireGuard handshake, and NAT'd internet egress — using two disposable, isolated Docker containers, without touching this machine's real networking at all. Already run successfully; see `local-test/README.md` and `docs/DECISIONS.md` 2026-08-26. This means the open items below are genuinely just "rent/configure real infrastructure," not "hope the design works."

## Before deploying anything for real

1. **Read LocalToNet's actual ToS/AUP** end to end — flagged as accepted risk in `docs/DECISIONS.md`, but "accepted risk" should mean "read and understood," not "unread." Not done yet. Still the single highest-priority item in `docs/OPEN_QUESTIONS.md`.
2. Pick which locations to launch with (`docs/OPEN_QUESTIONS.md`).
3. Pick a hosting provider for the 1-2 central servers (`docs/OPEN_QUESTIONS.md`).

## Current state (updated 2026-08-26)

`scripts/setup-central-server.sh` is a complete WireGuard setup: generates server keys, writes a real `wg0.conf` with NAT/forwarding rules (auto-detects the egress interface), persists IP forwarding, and enables a systemd service — not just a package install. Syntax-checked (`bash -n`) but **not run against a real VPS** — this environment only has containers, not a VM it can configure networking/systemd on. Review it before running on a real server, same as you would any infra script you didn't write yourself.

`LOCALTONET_SETUP.md` is a manual checklist (not a script) for the LocalToNet-account side, since that needs dashboard/account access this environment doesn't have.

Nothing has been provisioned. `backend/` (see `docs/BACKEND.md`) is fully implemented and ready to talk to a real central server the moment one exists — `WG_INTERFACE`/`DB_PATH` env vars and `internal/servers/locations.json` are the only things that need real values.
