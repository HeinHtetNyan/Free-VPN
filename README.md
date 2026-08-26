# SY VPN

Multi-location VPN: Android first, native iOS later without much rework. Free to use with ads; an optional paid tier may be added later (not decided yet).

This README is the map. Read `docs/` before making changes — it explains *why* things are structured this way, not just what's in each folder.

## Structure

```
.
├── docs/          Architecture, decisions, glossary — read this first
├── backend/       Go control plane: auth, user management, server list, WireGuard peer provisioning + registration
├── android/       Native Android app (Kotlin + Jetpack Compose), thin client — package com.syvpn.app
└── infra/         Central-server WireGuard setup, LocalToNet tunnel checklist, and a self-contained local test proving it all works (see docs/ARCHITECTURE.md for why it's central-server-based, not per-region VPS)
```

## Read these in order

1. [`docs/GLOSSARY.md`](docs/GLOSSARY.md) — terms used everywhere else in these docs
2. [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — the overall system: LocalToNet relays in front of 1-2 central servers, and why
3. [`docs/BACKEND.md`](docs/BACKEND.md) — control plane responsibilities and structure
4. [`docs/MOBILE.md`](docs/MOBILE.md) — Android app plan, and the iOS reuse strategy
5. [`docs/INFRA.md`](docs/INFRA.md) — central server + LocalToNet tunnel setup, and the local validation test
6. [`docs/MONETIZATION.md`](docs/MONETIZATION.md) — ads-supported free model (Adsterra), optional purchase later
7. [`docs/PLAY_STORE_COMPLIANCE.md`](docs/PLAY_STORE_COMPLIANCE.md) — VPN + ads policy risk checklist, read before shipping anything to the Play Store
8. [`docs/PRIVACY_POLICY_DRAFT.md`](docs/PRIVACY_POLICY_DRAFT.md) — first draft, needs review before publishing
9. [`docs/DECISIONS.md`](docs/DECISIONS.md) — decision log with rationale, so we don't relitigate settled questions
10. [`docs/OPEN_QUESTIONS.md`](docs/OPEN_QUESTIONS.md) — everything still pending on the user's side

## Paused here (2026-08-27) — resume with this

**Backend is live in production**, verified end-to-end for real (not just locally): `https://sy-api.heinh.dev/health` → `200`, and a real register→locations→connect smoke test produced a real peer on the production `wg0` (confirmed via `wg show`, cleaned up after). Everything buildable without external accounts is done. Pick up with:

**Done since 2026-08-26:**
- **Central server** — Shared VPS, project at `/home/appbox/SY/` (deliberately not VPN-named — see `docs/DECISIONS.md`). `wg0` up, listening UDP `51820`, server public key `4b3P37J2ZE3Hj2xDyCFPsFUWrJM+DZRCmE//5MRXdEo=`.
- **Backend deployed** — `backend/` running there (`docker compose`, `network_mode: host` + `cap_add: NET_ADMIN` for `wgctrl`'s netlink access to `wg0`), bound to `127.0.0.1:8080` only.
- **Public hostname** — `sy-api.heinh.dev` via a dedicated Cloudflare Tunnel (not `sawyuntech.com` — see `docs/DECISIONS.md` for why), live and serving real traffic.
- **LocalToNet connected** — account set up, first tunnel (Singapore) created, their client running as a systemd service on the same VPS with the real device token.
- **Adsterra wiring** — `android/` reads the zone ID from `BuildConfig.ADSTERRA_BANNER_ZONE_ID`, sourced from gitignored `android/local.properties`. Dropping a real zone ID in needs no code changes.
- **Deploy pipeline** — GitHub Actions workflow + `deploy.sh` exist and are pushed; first deploy was done manually over SSH rather than waiting on the secret below.

**Only the user can do these:**
1. **Confirm LocalToNet tunnel `2297029`'s Local IP/Port are `127.0.0.1`/`51820`, and get its assigned public relay port** once it shows connected — needed to replace the `REPLACE_WITH_LOCALTONET_ENDPOINT:PORT` placeholder in `backend/internal/servers/locations.json` with the real thing. This is the last piece between "backend works" and "a phone can actually connect."
2. Add `VPS_DEPLOY_KEY` as a GitHub repo secret (value already generated and handed over) so future pushes to `backend/**` auto-deploy instead of needing a manual SSH deploy each time.
3. **Adsterra** — sign up as publisher, create a Banner ad unit (not Popunder/Social Bar — see `docs/PLAY_STORE_COMPLIANCE.md`), hand back the zone ID. Drops straight into `android/local.properties`.
4. A real contact email for `docs/PRIVACY_POLICY_DRAFT.md`, and a decision on legal review before publishing.

**Claude can do without waiting on the above:**
- Get the Android app actually running on an emulator (not just compiling) — this machine has KVM available. Offered, not yet done.
- Point `android/data/ApiClient.kt`'s `DEV_BASE_URL` at `https://sy-api.heinh.dev` once real device testing (vs. emulator-only) is wanted.

## What's already done (for context when resuming)

The app is named **SY VPN** (`com.syvpn.app`). **The whole mechanism has been validated end-to-end locally** — not just "should work":

- **`backend/`** — fully implemented, tested (`go test ./...`), and containerized (`backend/Dockerfile`). Anonymous device-bound auth, SQLite persistence (verified across restarts), real WireGuard key generation, live peer registration via `wgctrl`. See `backend/README.md`.
- **`infra/local-test/`** — a self-contained Docker-based integration test proving the *entire* chain works: real keys → live peer registration → real WireGuard handshake → real NAT'd internet egress through the server. Two disposable, isolated containers; zero changes to this machine's actual networking. Already run successfully. See `infra/local-test/README.md`.
- **`android/`** — real Gradle/Kotlin/Compose project (connect flow, location picker, WireGuard tunnel integration, Adsterra ad component, placeholder app icon). **Compiles successfully** — verified via a Docker-based headless Android SDK build (`android/Dockerfile`), producing a real `app-debug.apk`. Not yet run on an emulator/device.
- **`infra/`** — `setup-central-server.sh` is a complete WireGuard+NAT+systemd setup (syntax-checked, not run against a real VPS, but the mechanism it sets up is exactly what `local-test/` already validated); `LOCALTONET_SETUP.md` is the manual checklist for the account side.
- **`docs/PRIVACY_POLICY_DRAFT.md`** — first draft written against actual app behavior, not yet reviewed or published.
- Nothing is live/public. No git operations have been done (explicit user instruction: don't push yet).
