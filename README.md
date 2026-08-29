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

**The entire app works, verified on a real phone over the real internet** — not an emulator, not local-test. Installed on the user's device via wireless `adb`, tapped Connect, granted the real Android VPN permission dialog, got a real WireGuard handshake through the real LocalToNet relay, and browsed the actual internet through it (confirmed via a real browser page load + the status bar's VPN key icon). This is the actual product working end to end.

Two real bugs found and fixed by this real-device test (would never have been caught by local-test or a code read):
1. `MainActivity.onConnectClick()` called `vpnManager.connect()` inside a nested coroutine `launch{}` that the enclosing `try/catch` didn't actually cover — a `BackendException`/`TimeoutException` from the WireGuard library crashed the app instead of showing an error.
2. The UI was setting `Connected` immediately after starting the connect flow, before the permission dialog was even answered — cosmetically "connected" while nothing had happened yet.
Both fixed in `MainActivity.kt`; `vpnPermissionLauncher`'s callback now also runs `connect()` on a background dispatcher with proper error handling instead of blocking the main thread uncaught.

Also fixed: each Docker build run was generating a fresh random debug-signing keystore, so a rebuilt APK couldn't be installed over the previous one (`INSTALL_FAILED_UPDATE_INCOMPATIBLE`) without a manual uninstall first. `android/app/debug.keystore` is now committed and pinned in `build.gradle.kts` — every build now signs identically.

Everything buildable without external accounts is done. Pick up with:

**Done since 2026-08-26:**
- **Central server** — Shared VPS, project at `/home/appbox/SY/` (deliberately not VPN-named — see `docs/DECISIONS.md`). `wg0` up, listening UDP `51820`, server public key `4b3P37J2ZE3Hj2xDyCFPsFUWrJM+DZRCmE//5MRXdEo=`.
- **Backend deployed** — `backend/` running there (`docker compose`, `network_mode: host` + `cap_add: NET_ADMIN` for `wgctrl`'s netlink access to `wg0`), bound to `127.0.0.1:8080` only.
- **Public hostname** — `sy-api.malmah.fyi` via a dedicated Cloudflare Tunnel (not `sawyuntech.com` — see `docs/DECISIONS.md` for why), live and serving real traffic.
- **LocalToNet connected** — account set up, first tunnel (Singapore) created, their client running as a systemd service on the same VPS with the real device token.
- **Adsterra wiring** — `android/` reads the zone ID from `BuildConfig.ADSTERRA_BANNER_ZONE_ID`, sourced from gitignored `android/local.properties`. Dropping a real zone ID in needs no code changes.
- **Deploy pipeline** — GitHub Actions workflow + `deploy.sh` exist and are pushed; first deploy was done manually over SSH rather than waiting on the secret below.

**Only the user can do these:**
1. Add `VPS_DEPLOY_KEY` as a GitHub repo secret (value already generated and handed over) — **verified working** 2026-08-27 (a real push triggered a real auto-deploy). Nothing further needed here, just noting it's genuinely done, not just set up.
2. **Adsterra** — sign up as publisher, create a Banner ad unit (not Popunder/Social Bar — see `docs/PLAY_STORE_COMPLIANCE.md`), hand back the zone ID. Drops straight into `android/local.properties`.
3. A real contact email for `docs/PRIVACY_POLICY_DRAFT.md`, and a decision on legal review before publishing.
4. Which additional locations to launch with beyond Singapore (needs more LocalToNet tunnels, same process as the first one).

**Claude can do without waiting on the above:**
- Nothing blocking left on the core mechanism — what remains (Adsterra creative/placement polish, Play Store listing, additional locations) all needs real accounts/decisions from the user first.

## What's already done (for context when resuming)

The app is named **SY VPN** (`com.syvpn.app`). **The whole mechanism has been validated end-to-end for real** — locally, in production, and on an actual phone over the real internet, not just "should work":

- **`backend/`** — fully implemented, tested (`go test ./...`), containerized, and **deployed to production** at `https://sy-api.malmah.fyi` (Shared VPS `/home/appbox/SY/`). Anonymous device-bound auth, SQLite persistence, real WireGuard key generation, live peer registration via `wgctrl` against the real `wg0`. See `backend/README.md`.
- **`infra/local-test/`** — a self-contained Docker-based integration test proving the mechanism works with zero changes to this dev machine's networking. Superseded in importance by the real production deployment below, but still useful for future local iteration. See `infra/local-test/README.md`.
- **Central WireGuard server** — live on the Shared VPS, `wg0` up on UDP `51820`, systemd-managed.
- **LocalToNet** — real account, tunnel connected (`vbznwgzpdl.localto.net:2020` → Singapore), their client running as a systemd service on the same VPS.
- **`android/`** — real Gradle/Kotlin/Compose project. **Installed and used on a real physical phone** via wireless `adb` (not an emulator): real connect flow, real Android VPN permission dialog, real WireGuard handshake through the real LocalToNet relay, real browsing through the tunnel. Two real bugs found and fixed by this test (see the top of this section) — the kind of thing no amount of code review or local-test would have caught.
- **`docs/PRIVACY_POLICY_DRAFT.md`** — first draft written against actual app behavior, not yet reviewed or published.
- Repo is pushed to GitHub (`HeinHtetNyan/Free-VPN`) and auto-deploys on push to `backend/**`.
