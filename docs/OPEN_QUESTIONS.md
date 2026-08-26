# Open questions

Things still pending, mostly on the user's side. Check this before assuming something is decided — cross-reference `docs/DECISIONS.md` for what's already settled.

## Branding / identity
- [x] ~~VPN app name~~ — resolved: **SY VPN**, package `com.syvpn.app`. Renamed everywhere (Android package physically moved, Go module renamed, `strings.xml` updated) and verified via a full Docker rebuild. See `docs/DECISIONS.md`.
- [ ] Real logo/visual branding — current icon is a generic placeholder shield mark, functional but not a designed logo.

## LocalToNet
- [x] ~~Read LocalToNet's actual ToS~~ — done: it prohibits this use case ("reselling, duplicating, or exploiting any part of LocalToNet without written permission"). See `docs/DECISIONS.md` 2026-08-26.
- [x] ~~Decide how to proceed~~ — resolved: proceed with LocalToNet anyway, knowingly accepting the ToS violation risk (account suspension would take every location down at once). User's explicit, fully-informed choice. See `docs/DECISIONS.md`.
- [ ] Which locations to launch with (Thailand isn't directly in their region list — nearest options are Singapore/Hong Kong; confirm what's actually available in their dashboard).
- [ ] Account/plan set up (paid tier needed — free tier's 1GB/month + 30-min timeout is unusable here).

## Backend
- [x] ~~Auth strategy~~ — resolved: anonymous device-bound identity. Implemented in `backend/internal/auth`, `backend/internal/users`. See `docs/DECISIONS.md`.
- [x] ~~Database choice~~ — resolved: SQLite (`modernc.org/sqlite`), implemented and tested (persists across restarts). See `docs/BACKEND.md`.
- [x] ~~Does the whole mechanism actually work?~~ — resolved: yes, fully validated locally with zero host networking changes. See `infra/local-test/` and `docs/DECISIONS.md` 2026-08-26.
- [ ] Hosting provider for the 1-2 central servers — needed to actually run `infra/scripts/setup-central-server.sh` and get real Endpoint/public-key values into `backend/internal/servers/locations.json` and the `SERVER_PUBLIC_KEY` env var. This is the last remaining backend blocker — the mechanism is proven, the code is tested; what's missing is a real machine to run it on.

## Ads (Adsterra)
- [ ] Adsterra publisher account + zone IDs.
- [ ] Exact ad placement in the app flow (see `docs/MONETIZATION.md`).

## Monetization (paid tier)
- [ ] Whether/when to add a paid ad-free tier, and how (Google Play Billing is the natural fit for Android). Not designed yet — explicitly deferred.

## Play Store compliance
- [x] ~~Privacy Policy — first draft~~ — see `docs/PRIVACY_POLICY_DRAFT.md`. Still needs: real placeholders filled in, legal review, and actually publishing it somewhere with a stable URL before it satisfies the Play Store requirement.
- [ ] Play Console Data Safety form — fill out once auth + Adsterra integration are final.
- [ ] Re-check `docs/PLAY_STORE_COMPLIANCE.md` against Google's current policy text before first submission.

## Future / not urgent
- [ ] DPI/censorship resistance — plain WireGuard is fingerprintable and blockable; Myanmar's filtering is known to do this. Evaluate DPI-resistant options (e.g. AmneziaWG) once the core app works end-to-end. See `docs/ARCHITECTURE.md`.
- [ ] Whether to eventually migrate specific locations from LocalToNet relay to a dedicated VPS (see `docs/ARCHITECTURE.md` "Revisit if...").
