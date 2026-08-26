# Decision log

Newest first. Each entry: what was decided, why, and what it rules out — so we don't relitigate settled questions without new information.

---

### 2026-08-27 — Verified on a real physical device over wireless adb; two real bugs found and fixed

User asked to install the app on their own phone rather than an emulator. Used wireless `adb` (already paired, auto-discovered via `adb mdns services`) — no cable, no emulator. This is meaningfully stronger verification than `infra/local-test/`: it exercises the real `VpnService`/`GoBackend` Android integration and the real system permission dialog, neither of which a Docker container touches.

First real end-to-end result: a genuine WireGuard handshake through the real LocalToNet relay, with actual growing byte counters on both sides, and a real page load through the tunnel confirmed in a browser (status bar VPN key icon present). But getting there required fixing two real bugs the app shipped with, caught only by this real test:

1. **Crash on connect**: `MainActivity.onConnectClick()` called the blocking `vpnManager.connect()` inside a nested coroutine `launch{}` whose exceptions weren't covered by the enclosing `try/catch` (a `launch{}` starts an independent child coroutine — an outer `try` around the `launch` call doesn't catch what happens inside it). A `BackendException`/`TimeoutException` from the WireGuard library therefore crashed the whole app instead of surfacing a UI error. Fixed by keeping the risky call inside the same coroutine body as its own `try/catch`, only switching to `Dispatchers.Main` for UI-only work (launching the permission intent, updating state).
2. **Premature "Connected" state**: the UI set `ConnectionUiState.Connected` immediately after starting the connect flow, even in the branch that still needed the user to answer the system VPN permission dialog — so the screen showed "Connected via singapore" before the dialog had even been answered.

Also fixed as a side effect of doing real rebuild-and-reinstall cycles: each Docker build run was generating a **fresh random debug-signing keystore** (no persisted `~/.gradle`/`~/.android` across ephemeral container runs), so a rebuilt APK couldn't be installed over the previous one without `adb uninstall` first (`INSTALL_FAILED_UPDATE_INCOMPATIBLE`). Generated and committed `android/app/debug.keystore`, pinned explicitly in `build.gradle.kts`'s `signingConfigs.debug` — every build now signs identically. Standard well-known debug alias/password, not a secret.

### 2026-08-27 — Backend public hostname: sy-api.heinh.dev, not sawyuntech.com

Needed a domain for the backend's Cloudflare Tunnel. Deliberately did not reuse `sawyuntech.com` (already used for other apps, but it's the Saw Yun LLC company domain — reusing it would undercut the earlier app-naming decision to keep this project disconnected from that brand). User's own `heinh.dev` (already used for TK Plastic Press/BonBon/code-server, unrelated to any company brand) instead. Tunnel created via API (id `97787a8f-a3e5-4a01-9d19-797e843790da`), DNS record `sy-api.heinh.dev` → proxied CNAME to the tunnel, ingress routes to `http://localhost:8080` (matches `backend/`'s default `PORT`). Not yet running — `tunnel/.env` has the real token locally, waiting on the repo being pushed/cloned onto the VPS.

### 2026-08-27 — Central server hosting: the existing Shared VPS, project kept under a non-VPN folder name

Used the already-owned Shared VPS (`hhn.infinity.appboxes.co`, see memory) instead of provisioning a new box — no extra cost, and it already runs several other unrelated apps (BonBon POS, RateBridge, n8n, a portfolio site) under isolated Docker networks. Project files placed at `/home/appbox/SY/`, deliberately not named anything VPN-related — same operational-precaution reasoning as the `com.syvpn.app` naming decision below, though it's a light touch: the folder name doesn't hide the running `wg0` interface, listening UDP port, or `wireguard` package from anyone who actually inspects the host (there just isn't anyone else with shell access to this box to notice).

Ran `infra/scripts/setup-central-server.sh` for real: `wg0` up, `10.66.0.1/16`, listening UDP `51820`, systemd `wg-quick@wg0` enabled (survives reboot). Server public key: `4b3P37J2ZE3Hj2xDyCFPsFUWrJM+DZRCmE//5MRXdEo=`. Checked first for conflicts with the box's other tenants: no existing WireGuard install, port 51820 free, no Docker subnet overlap with `10.66.0.0/16` (existing networks are all `172.x`), `ip_forward` was already `1` (Docker had already set it — the script's sysctl step was a no-op). Still open: deploying `backend/` itself to this host (not done yet — this pass was WireGuard only) and setting `SERVER_PUBLIC_KEY` in its environment when that happens, per the env-var mechanism from the 2026-08-26 rename entry below.

### 2026-08-26 — LocalToNet's actual ToS read: it prohibits this use case

This was the single highest-priority open item throughout this whole build (`docs/OPEN_QUESTIONS.md`) — finally read it. Their Terms of Use (`localtonet.com/terms`, confirmed via two independent fetches) state: *"You shall not use LocalToNet for any inappropriate purpose, such as reselling, duplicating, or exploiting any part of LocalToNet without written permission."* Running a commercial VPN product's traffic backbone through their tunnels is a straightforward case of "reselling/exploiting" their service. This is **not** a stretch interpretation.

This reverses the earlier "use it for production too, accept the risk" decision (below) — that decision was made *before* reading the actual terms, on the (reasonable at the time) assumption the risk was real-but-fuzzy. It is now a confirmed, explicit prohibition, not a fuzzy risk. The architecture built throughout this session (`docs/ARCHITECTURE.md`) still works mechanically — proven by `infra/local-test/` — but running it against LocalToNet without written permission from them is now a known ToS violation, not a calculated risk.

**Resolved same day: proceed with LocalToNet anyway, knowingly.** User's explicit choice after being presented with the ToS finding and all three options (contact them for written permission, switch to dedicated per-region VPS, proceed accepting the risk). This is now a deliberate, fully-informed decision — not the earlier "accepted the risk without having read the terms" position. The risk itself is unchanged: an account suspension would take every location down simultaneously, since all locations funnel through the same 1-2 central servers. Nothing about the technical design changes as a result of this decision.

### 2026-08-26 — App named: SY VPN. Full rename pass done.

Chosen deliberately disconnected from the company name (Saw Yun LLC) — reasonable operational precaution for a censorship-circumvention tool. Renamed everywhere: Android package `com.placeholder.vpnapp` → `com.syvpn.app` (files physically moved, not just find-replaced — package declarations and imports updated), Gradle project name → `sy-vpn`, Go module `vpnapp-backend` → `sy-vpn-backend` (all internal imports updated), app display name → "SY VPN" in `strings.xml`. Verified via a full Docker rebuild after the rename — still `BUILD SUCCESSFUL`. Also made the central server's public key configurable via a `SERVER_PUBLIC_KEY` env var (`api.DefaultServerPublicKeyPlaceholder` as the fallback) instead of a hardcoded Go constant, matching the existing `WG_INTERFACE`/`DB_PATH` pattern — this was needed for the local end-to-end test below, and is also just a real improvement (no more code edit + rebuild needed to set the real server key later).

### 2026-08-26 — Full mechanism validated locally, with zero host networking changes (`infra/local-test/`)

User asked to set up and test everything locally before moving to real server hosting. Built a fully self-contained Docker-based integration test (`infra/local-test/run.sh`) proving the entire chain works: real backend code → real WireGuard Curve25519 key generation → live peer registration via `wgctrl` against a real kernel WireGuard interface → real encrypted WireGuard handshake → (in the default, non-`--scoped` mode) real NAT/`ip_forward` relay of traffic through the "server" container to the actual internet. Verified: `wg show` showed a completed handshake with non-zero transfer bytes; ping across the tunnel succeeded; an external HTTP fetch succeeded *only because* the client's entire default route had been captured by the tunnel interface, which is real evidence the server-side NAT path works, not a mock.

Deliberately used two isolated, ephemeral Docker containers on a private Docker network (`--network bridge`, not `--network host`) rather than touching this machine's actual network state — no interfaces, iptables rules, or sysctls were changed on the host itself. This mattered because this environment turned out to not be a disposable sandbox (other unrelated containers were already running here), so host-level network changes would have needed to be explicitly confirmed first; the fully-containerized approach avoided needing that conversation at all while still proving the same thing.

Two Docker-specific frictions hit and resolved, both **artifacts of testing inside a container, not of the actual design**: (1) `sysctl -w` fails at runtime inside a container even with `NET_ADMIN` — namespaced sysctls like `net.ipv4.ip_forward` must be set via `docker run --sysctl` at container start instead; (2) full default-route capture (`AllowedIPs=0.0.0.0/0`) needs `net.ipv4.conf.all.src_valid_mark=1`, which stays read-only even via `--sysctl` for non-privileged containers — required `--privileged` (still fully isolated, no `--network host`) to test the complete production-equivalent config. Neither issue will exist on the real central server, which runs WireGuard natively on a VPS with full root access, not inside Docker — see `infra/local-test/README.md` "Why this doesn't (and shouldn't) mirror production exactly."

This means: the design is now proven correct end-to-end, independent of and before spending anything on LocalToNet or VPS hosting. What's left really is just plugging real infrastructure into already-working, already-tested code.

### 2026-08-26 — Everything buildable without server hosting / LocalToNet account / Adsterra account, pushed to completion

User asked to finish the build to the end, leaving only the three externally-blocked pieces (central server hosting, LocalToNet account, Adsterra account) for later. Closed out everything else that was still open:

- **SQLite persistence** implemented for real (`internal/users`, `internal/servers/peer_store.go`) — verified surviving a full server restart, not just "should work."
- **Live WireGuard peer registration** wired up via `wgctrl` (`internal/servers/wireguard_manager.go`), called from `handleConnect`. Best-effort by design (logs a warning and still returns the config) since no real WireGuard interface exists yet — see `docs/BACKEND.md` for the TODO to make this a hard failure once a central server is live.
- **`infra/scripts/setup-central-server.sh`** completed into a real, complete WireGuard+NAT+systemd setup (was previously just a package-install stub), plus `infra/LOCALTONET_SETUP.md` for the manual account-side steps.
- **`backend/Dockerfile`** added and verified (built, ran, hit its endpoints in-container) — a deployable artifact for whenever hosting exists.
- **Backend test suite** added (`go test ./...`, all passing) covering the full register→locations→connect flow, idempotency, persistence, and error paths.
- **Android app icon** — a placeholder adaptive icon (shield+checkmark) replacing Android's default, verified via a full Docker rebuild.
- **Privacy Policy first draft** (`docs/PRIVACY_POLICY_DRAFT.md`) — written against what the app actually does, not a generic template; still needs legal review and publishing before it satisfies the Play Store requirement.

What's left is exactly the three things the user named: central server hosting, a LocalToNet account, an Adsterra account — plus whatever downstream config (real endpoints, zone IDs) only exists once those are set up.

### 2026-08-26 — Android build verified via Docker (no Android Studio needed for this)

This dev environment has no Android SDK/Studio, but does have Docker. Built `android/Dockerfile` (JDK17 + Android cmdline-tools + platform 35 + build-tools) and used it to run `./gradlew assembleDebug` for real — `BUILD SUCCESSFUL`, produced a working `app-debug.apk`. This caught a real bug: Kotlin 2.0+ needs the separate `org.jetbrains.kotlin.plugin.compose` Gradle plugin, missing from the initial scaffold — fixed in both `build.gradle.kts` files. Also generated the actual Gradle wrapper (`gradlew`/`gradlew.bat`/`gradle-wrapper.jar`) via the official `gradle:8.9-jdk17` image rather than leaving it for Android Studio to bootstrap. This verifies the code *compiles*, not that the UI/tunnel behave correctly at runtime — that still needs an emulator or device, which is a heavier setup (display, adb) not attempted yet. See `android/README.md` for the exact commands to reproduce.

### 2026-08-26 — Backend implemented: anonymous device-bound auth, in-memory store for now

Built and tested (`go build` + manual curl flow) `POST /auth/register`, `GET /locations`, `POST /connect` against `backend/`. Resolved the previously-open auth-strategy question in favor of anonymous device-bound identity over real accounts — matches the frictionless-onboarding UX priority, and avoids asking for an email up front on what's substantially a censorship-circumvention tool. Accepted tradeoff: no cross-device subscription restore without adding real accounts later (fine for now, paid tier isn't designed yet). Database is intentionally in-memory for this pass, not yet SQLite — kept the backend dependency-light while the rest of the stack (LocalToNet account, central server) still doesn't exist; swap before real usage. `/connect` generates real WireGuard keys but returns placeholder Endpoint/server-public-key values since no central server is deployed yet. Go itself wasn't installed in the dev environment — installed user-locally to `~/go-sdk` (no sudo available/needed).

### 2026-08-26 — Play Store policy compliance is a design constraint, not a pre-launch checklist

Consolidated into `docs/PLAY_STORE_COMPLIANCE.md`. Concretely changes two earlier decisions from "nice to have" to "required": (1) the transparent-location-labeling decision below is now also a Deceptive Behavior policy requirement, not just UX; (2) the Adsterra format restriction (Banner/Native/Interstitial, no Popunder/Social Bar) is now also a policy requirement, not just a UX preference. Also adds new requirements not previously tracked: a real Privacy Policy disclosing LocalToNet's metadata visibility and Adsterra's data collection, and a Play Console Data Safety form that must match actual behavior.

### 2026-08-26 — Ad network: Adsterra

Checked their actual offering: no native Android SDK — it's webview/JS-tag based (Banner, Native, Interstitial, Social Bar, Popunder, In-Page Push, Smartlink). Decided to use Banner/Native/Interstitial via a contained WebView; ruled out Popunder/Social Bar for the shipped app specifically because those formats commonly trigger Play Store policy rejections (ads that interfere with/mimic system UI). See `docs/MONETIZATION.md`.

### 2026-08-26 — Target audience is Myanmar users

Changes how much the "location = cosmetic, not real geo-spoofing" tradeoff (below) matters: primary goal is reaching the open internet from behind local network restrictions, not spoofing a specific country for streaming/geo-unlock. Also surfaces a new concern not previously tracked: Myanmar's network filtering can use DPI to detect and block VPN protocols outright, including plain WireGuard. Tracked as an open question, not yet solved. See `docs/ARCHITECTURE.md` "Censorship resistance."

### 2026-08-26 — Location selection will be UI-transparent, not implied geo-spoofing

Since exit IP is always the central server's real location regardless of which relay/location the user picks (see next entry), the app will frame location choice honestly (e.g. "fastest connection point") instead of implying the user will appear to browse from that country. Considered and rejected: co-locating central servers with key target locations to make some locations "really" correct — deferred, not ruled out permanently, since Myanmar-focused use case reduces how much this matters right now.

### 2026-08-26 — Use LocalToNet + 1-2 central servers in production, accepting the tradeoffs

Reversed an earlier lean (dedicated per-region VPS exit nodes) after checking LocalToNet's actual offering: it does have real regions relevant here (Singapore, Tokyo, Hong Kong, several in Europe) and supports UDP tunnels (WireGuard needs UDP), at $2/mo per tunnel with unlimited bandwidth on the paid plan — cheaper than a VPS per region.

Knowingly accepted in exchange for that cost savings:
- **ToS risk** — LocalToNet markets itself for dev-tunnel/webhook/game-server use, not reselling VPN service; their actual ToS/AUP has not been read yet (action item, see `docs/OPEN_QUESTIONS.md`). Violation risk = every location suspended at once, since all locations funnel through the same 1-2 central servers.
- **Metadata exposure** to LocalToNet's relays (source IP, timing, volume per session), even though the WireGuard payload stays encrypted.
- **Single point of failure** — all locations go down together if the central server(s) or LocalToNet has an outage, unlike independently-failing dedicated regional nodes.
- **Exit IP is not really per-location** — see entry above.

This can be revisited later (migrate specific high-traffic locations to dedicated VPS) without a rewrite, since the control plane treats "location -> relay/server mapping" as data, not something hardcoded into the app.

### 2026-08-26 — Backend language: Go

Pairs natively with WireGuard tooling (`wgctrl-go`), and control plane + WireGuard now run on the same central server(s), so a single Go binary can both serve the API and shell out to manage WireGuard peers directly — no separate remote-management protocol needed.

### 2026-08-26 — Mobile: native Android now (Kotlin + Compose), thin client; native iOS later

Rejected Flutter (WireGuard plugins are community-maintained, not official) and Kotlin Multiplatform (unnecessary complexity) in favor of putting business logic in the backend API instead of shared mobile code. iOS later just means building iOS UI + using WireGuard's official iOS library against the same API.

### 2026-08-26 — Monetization: free with ads now; optional paid tier later, undecided shape

Ad network and placement details to be provided later. Backend models user tier as a simple flag rather than building billing infrastructure ahead of need.

### 2026-08-26 — Project name: placeholder pending

App/VPN name and branding not yet chosen. Scaffolding uses a generic placeholder package ID / project name, deliberately kept to a single find-and-replace when the real name arrives (tracked in `docs/OPEN_QUESTIONS.md`).
