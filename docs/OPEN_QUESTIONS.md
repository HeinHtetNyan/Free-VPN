# Open questions

Things still pending, mostly on the user's side. Check this before assuming something is decided — cross-reference `docs/DECISIONS.md` for what's already settled.

## Branding / identity
- [x] ~~VPN app name~~ — resolved: **SY VPN**, package `com.syvpn.app`. Renamed everywhere (Android package physically moved, Go module renamed, `strings.xml` updated) and verified via a full Docker rebuild. See `docs/DECISIONS.md`.
- [ ] Real logo/visual branding — current icon is a generic placeholder shield mark, functional but not a designed logo.

## LocalToNet
- [x] ~~Read LocalToNet's actual ToS~~ — done: it prohibits this use case ("reselling, duplicating, or exploiting any part of LocalToNet without written permission"). See `docs/DECISIONS.md` 2026-08-26.
- [x] ~~Decide how to proceed~~ — resolved: proceed with LocalToNet anyway, knowingly accepting the ToS violation risk (account suspension would take every location down at once). User's explicit, fully-informed choice. See `docs/DECISIONS.md`.
- [x] ~~Account/plan set up~~ — done 2026-08-26: first tunnel created (id `2297029`, relay `vbznwgzpdl.localto.net`, server `sg2`/Singapore), their client installed and running as a systemd service on the Shared VPS with the real device AuthToken — see `infra/LOCALTONET_SETUP.md`.
- [ ] Which additional locations to launch with (Thailand isn't directly in their region list — nearest options are Singapore/Hong Kong; confirm what's actually available in their dashboard). One (Singapore) exists so far.
- [ ] Confirm tunnel `2297029`'s Local IP/Port are set to `127.0.0.1`/`51820`, and note the relay's assigned public port once the tunnel shows connected — both needed for `backend/internal/servers/locations.json`.

## Backend
- [x] ~~Auth strategy~~ — resolved: anonymous device-bound identity. Implemented in `backend/internal/auth`, `backend/internal/users`. See `docs/DECISIONS.md`.
- [x] ~~Database choice~~ — resolved: SQLite (`modernc.org/sqlite`), implemented and tested (persists across restarts). See `docs/BACKEND.md`.
- [x] ~~Does the whole mechanism actually work?~~ — resolved: yes, fully validated locally with zero host networking changes. See `infra/local-test/` and `docs/DECISIONS.md` 2026-08-26.
- [x] ~~Hosting provider for the 1-2 central servers~~ — resolved 2026-08-27: using the existing Shared VPS (`hhn.infinity.appboxes.co`), project files kept under the deliberately non-VPN-named `/home/appbox/SY/` (see `docs/DECISIONS.md`). `infra/scripts/setup-central-server.sh` has been run there: `wg0` up, listening UDP `51820`, server public key `4b3P37J2ZE3Hj2xDyCFPsFUWrJM+DZRCmE//5MRXdEo=`. Still open: deploying `backend/` itself to this host and setting `SERVER_PUBLIC_KEY` in its environment when that happens.

## Ads (Adsterra)
- [ ] Adsterra publisher account + zone IDs.
- [ ] Exact ad placement in the app flow (see `docs/MONETIZATION.md`).

## Backend hosting/deploy (updated 2026-08-26)
- [x] ~~Public hostname for the backend~~ — resolved: `sy-api.heinh.dev`, via a dedicated Cloudflare Tunnel (id `97787a8f-a3e5-4a01-9d19-797e843790da`) on the existing `heinh.dev` zone/account, not `sawyuntech.com` (keeps it off the Saw Yun LLC brand, per the app-naming decision). DNS + tunnel config created via API; `tunnel/.env`'s token is in place locally, not yet running on the VPS.
- [ ] Push the repo to GitHub (`origin` = `HeinHtetNyan/Free-VPN`), add `VPS_DEPLOY_KEY` as a repo secret, then clone onto the Shared VPS at `/home/appbox/SY/` and run the first deploy — everything else is ready and waiting on this.

## Monetization (paid tier)
- [ ] Whether/when to add a paid ad-free tier, and how (Google Play Billing is the natural fit for Android). Not designed yet — explicitly deferred.

## Play Store compliance
- [x] ~~Privacy Policy — first draft~~ — see `docs/PRIVACY_POLICY_DRAFT.md`. Still needs: real placeholders filled in, legal review, and actually publishing it somewhere with a stable URL before it satisfies the Play Store requirement.
- [ ] Play Console Data Safety form — fill out once auth + Adsterra integration are final.
- [ ] Re-check `docs/PLAY_STORE_COMPLIANCE.md` against Google's current policy text before first submission.

## Future / not urgent
- [ ] DPI/censorship resistance — plain WireGuard is fingerprintable and blockable; Myanmar's filtering is known to do this. Evaluate DPI-resistant options (e.g. AmneziaWG) once the core app works end-to-end. See `docs/ARCHITECTURE.md`.
- [ ] Whether to eventually migrate specific locations from LocalToNet relay to a dedicated VPS (see `docs/ARCHITECTURE.md` "Revisit if...").
