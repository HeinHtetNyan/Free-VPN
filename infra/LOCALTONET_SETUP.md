# LocalToNet tunnel setup (manual — needs your account)

> **⚠ Known, accepted risk.** LocalToNet's Terms of Use prohibit this use case ("reselling, duplicating, or exploiting any part of LocalToNet without written permission") — confirmed by reading them directly (2026-08-26). The decision was made to proceed anyway, knowingly: see `../docs/DECISIONS.md`. That means an account suspension is a real possibility, not a hypothetical, and would take every location down at once (all locations funnel through the same 1-2 central servers). Worth having a plan for that scenario before launch — at minimum, know in advance what you'd do if the account got suspended with no warning.

Unlike `scripts/setup-central-server.sh`, this can't be scripted end-to-end from here — it needs your LocalToNet account/dashboard, which this environment has no access to. This is the checklist for once you're ready to do it.

## Steps

1. Run `scripts/setup-central-server.sh` on your central VPS first — you need its public IP and the WireGuard listen port (`WG_PORT`, default `51820`/UDP) to point a tunnel at.
2. Sign up for a **paid** LocalToNet plan — the free tier (1GB/month, 30-min timeout) can't sustain real VPN traffic (see `../docs/ARCHITECTURE.md`).
3. In the LocalToNet dashboard, create one **UDP tunnel per location** you want to advertise (e.g. Singapore, Frankfurt):
   - Region: the location this tunnel represents.
   - Forward-to: your central VPS's IP and `WG_PORT`.
   - Note the relay address (host:port) LocalToNet gives you back for each tunnel — that's what goes into the next step.
4. For each location, update `../backend/internal/servers/locations.json`:
   - `relay_address`: the LocalToNet relay endpoint from step 3.
   - `id`/`display_name`: whatever the app should show.
5. Restart `backend/` so it picks up the new `locations.json`.
6. Test end-to-end: register a device, call `/connect` for that location, and actually try the returned WireGuard config from a phone/laptop.

## Reminder

Exit IP will be your central VPS's location, not the advertised location's — that's the accepted tradeoff, not a bug. See `../docs/ARCHITECTURE.md` "What 'location' means here."
