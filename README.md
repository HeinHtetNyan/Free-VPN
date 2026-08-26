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

Work is paused pending the user setting up external accounts. **Nothing about the design or code is in question** — everything buildable without external accounts is done and verified. Pick up with:

**Done since 2026-08-26:**
- **Central server** — using the existing Shared VPS, project kept at `/home/appbox/SY/` on it (deliberately not VPN-named — see `docs/DECISIONS.md` 2026-08-27). `infra/scripts/setup-central-server.sh` run for real: `wg0` up, listening UDP `51820`, server public key `4b3P37J2ZE3Hj2xDyCFPsFUWrJM+DZRCmE//5MRXdEo=`. `backend/` itself is not deployed there yet.
- **Adsterra wiring** — `android/` now reads the zone ID from `BuildConfig.ADSTERRA_BANNER_ZONE_ID`, sourced from a gitignored `android/local.properties` (see `android/local.properties.example`). Dropping a real zone ID in there needs no further code changes. Verified via a full Docker rebuild.

**Only the user can do these (accounts/payment):**
1. **LocalToNet** — sign up (paid plan; free tier is unusable — see `docs/ARCHITECTURE.md`), create one UDP tunnel per launch location, each pointed at `127.0.0.1:51820` on the central VPS (their client runs on the VPS itself — see `infra/LOCALTONET_SETUP.md` for the corrected step-by-step). Hand back the relay addresses. Note: their ToS prohibits this use case and the decision was made to proceed anyway, knowingly — see `docs/DECISIONS.md` 2026-08-26. **Ignore their "VPN Manager" feature — that's a separate, unrelated product.**
2. **Adsterra** — sign up as publisher, create a Banner ad unit (not Popunder/Social Bar — see `docs/PLAY_STORE_COMPLIANCE.md`), hand back the zone ID. Drops straight into `android/local.properties`.
3. A real contact email for `docs/PRIVACY_POLICY_DRAFT.md`, and a decision on legal review before publishing.

**Claude can do without waiting on the above:**
- Get the Android app actually running on an emulator (not just compiling) — this machine has KVM available. Offered, not yet done.
- Deploy `backend/` to the Shared VPS (`/home/appbox/SY/`) once the user wants it live — the WireGuard side is already up and waiting.

## What's already done (for context when resuming)

The app is named **SY VPN** (`com.syvpn.app`). **The whole mechanism has been validated end-to-end locally** — not just "should work":

- **`backend/`** — fully implemented, tested (`go test ./...`), and containerized (`backend/Dockerfile`). Anonymous device-bound auth, SQLite persistence (verified across restarts), real WireGuard key generation, live peer registration via `wgctrl`. See `backend/README.md`.
- **`infra/local-test/`** — a self-contained Docker-based integration test proving the *entire* chain works: real keys → live peer registration → real WireGuard handshake → real NAT'd internet egress through the server. Two disposable, isolated containers; zero changes to this machine's actual networking. Already run successfully. See `infra/local-test/README.md`.
- **`android/`** — real Gradle/Kotlin/Compose project (connect flow, location picker, WireGuard tunnel integration, Adsterra ad component, placeholder app icon). **Compiles successfully** — verified via a Docker-based headless Android SDK build (`android/Dockerfile`), producing a real `app-debug.apk`. Not yet run on an emulator/device.
- **`infra/`** — `setup-central-server.sh` is a complete WireGuard+NAT+systemd setup (syntax-checked, not run against a real VPS, but the mechanism it sets up is exactly what `local-test/` already validated); `LOCALTONET_SETUP.md` is the manual checklist for the account side.
- **`docs/PRIVACY_POLICY_DRAFT.md`** — first draft written against actual app behavior, not yet reviewed or published.
- Nothing is live/public. No git operations have been done (explicit user instruction: don't push yet).
