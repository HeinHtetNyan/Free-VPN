# LocalToNet tunnel setup (manual — needs your account)

> **⚠ Known, accepted risk.** LocalToNet's Terms of Use prohibit this use case ("reselling, duplicating, or exploiting any part of LocalToNet without written permission") — confirmed by reading them directly (2026-08-26). The decision was made to proceed anyway, knowingly: see `../docs/DECISIONS.md`. That means an account suspension is a real possibility, not a hypothetical, and would take every location down at once (all locations funnel through the same 1-2 central servers). Worth having a plan for that scenario before launch — at minimum, know in advance what you'd do if the account got suspended with no warning.

Unlike `scripts/setup-central-server.sh`, this can't be scripted end-to-end from here — it needs your LocalToNet account/dashboard, which this environment has no access to. This is the checklist for once you're ready to do it. Steps below verified against LocalToNet's own docs (`localtonet.com/documents/udp`, their download page, and their "How To Use Localtonet" guide) on 2026-08-26.

**Ignore "VPN Manager" in their dashboard** — it's a separate, unrelated product (their own mesh-VPN service, with its own client app and virtual IPs). We use a plain **UDP tunnel** instead, which just forwards raw UDP packets — that's what WireGuard needs.

## How it actually works (important correction)

LocalToNet's tunnel isn't dashboard-side "forward to my VPS's public IP." Instead: you install *their* client app **on the central VPS itself**, authenticated with a device-specific AuthToken. That client makes an *outbound* connection to a LocalToNet relay, and the tunnel's "local target" is `127.0.0.1:WG_PORT` — i.e. localhost on the same VPS, reachable by their client because it's running right there. The relay is what gets the public host:port you hand to end users.

## Steps

1. Run `scripts/setup-central-server.sh` on your central VPS first — you need `WG_PORT` (default `51820`/UDP) running there before the tunnel has anything to point at.
2. Sign up at `localtonet.com/register`, then pick a **paid** plan — the free tier (1GB/month, 30-min timeout, per earlier research) can't sustain real VPN traffic. Check current limits on their pricing page since these numbers can change (see `../docs/ARCHITECTURE.md`).
3. Dashboard → **My Tokens** → create a token for this VPS (e.g. name it `central-vps`). Copy it — you'll need it in the next step and it won't be shown again in full.
4. SSH into the central VPS and install the LocalToNet client as a systemd service:
   ```
   curl -fsSL https://localtonet.com/install.sh | sh
   sudo localtonet --install-service --authtoken <TOKEN_FROM_STEP_3>
   sudo localtonet --start-service --authtoken <TOKEN_FROM_STEP_3>
   ```
   Verify it's running: `systemctl status localtonet` / `journalctl -u localtonet -f`.
5. Dashboard → **TCP-UDP** page → create one tunnel **per location** you want to advertise (e.g. Singapore, Frankfurt), all using the *same* AuthToken from step 3 (one client, multiple tunnel configs):
   - Protocol: **UDP**
   - AuthToken: the `central-vps` token
   - Server/Relay: the region this tunnel represents
   - Local IP: `127.0.0.1`
   - Local Port: `WG_PORT` (`51820`)
   - Click **Create**, then **Start** — LocalToNet separates the two; a created-but-not-started tunnel has no public endpoint.
   - Note the public relay address (host:port) shown once it's running — that's what goes into the next step.
6. For each location, update `../backend/internal/servers/locations.json`:
   - `relay_address`: the LocalToNet relay endpoint from step 5.
   - `id`/`display_name`: whatever the app should show.
7. Restart `backend/` so it picks up the new `locations.json`.
8. Test end-to-end: register a device, call `/connect` for that location, and actually try the returned WireGuard config from a phone/laptop.

## Reminder

Exit IP will be your central VPS's location, not the advertised location's — that's the accepted tradeoff, not a bug. See `../docs/ARCHITECTURE.md` "What 'location' means here."
