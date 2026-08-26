# Architecture

> **⚠ 2026-08-26 update: LocalToNet's Terms of Use prohibit this use case** ("reselling, duplicating, or exploiting any part of LocalToNet without written permission"). Read in full, confirmed via two independent sources — not a fuzzy risk. **Decision: proceed anyway**, knowingly accepting that an account suspension would take every location down simultaneously (all locations funnel through the same 1-2 central servers). This was a deliberate, fully-informed choice, not an oversight — see `docs/DECISIONS.md` 2026-08-26 for both entries. The mechanism itself is proven working (`infra/local-test/`); this note exists so the risk stays visible rather than getting lost as "just an implementation detail."

## Decision: LocalToNet + 1–2 central servers (not per-region VPS)

We evaluated two approaches (see `docs/DECISIONS.md` for the full reasoning) and chose the LocalToNet-backed one for cost reasons, accepting the tradeoffs below deliberately.

```
                    Android app (later: iOS app)
                               │
                    HTTPS to control plane API
              (auth, get location list, request a
               WireGuard config for a location)
                               │
                               ▼
              Control plane + WireGuard server
                  (the same 1-2 central VPS)
                               │
              exposed to the internet as several
              WireGuard (UDP) tunnels via LocalToNet,
              one tunnel per advertised "location"
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
   LocalToNet             LocalToNet             LocalToNet
   Singapore relay        Tokyo relay            Frankfurt relay
        │                      │                      │
        └──────────────────────┼──────────────────────┘
                               ▼
                 forwarded back to the same central
                    WireGuard server + decrypted
                               │
                               ▼
                        real internet
        (exits using the central server's actual IP/
         location — NOT the relay's country. See below.)
```

## What "location" means here

The Android app will show a list of locations (Thailand, Singapore, Europe, ...). Each one maps to a LocalToNet relay endpoint in that region, which forwards the WireGuard (UDP) connection back to whichever central server is handling it. **Picking a location changes which relay the phone connects through (a possible latency/reachability benefit), not where the traffic exits to the internet.** Exit IP is always the central server's real location.

Decided: the UI will be transparent about this (e.g. framed as "fastest connection point," not "browse as if you're in this country"), rather than implying real geo-spoofing. See `docs/MONETIZATION.md` / `docs/MOBILE.md` for how this gets worded.

Since the primary use case is Myanmar users reaching the open internet from behind local network restrictions — not geo-unlocking foreign streaming catalogs — this gap matters less than it would for a typical consumer VPN. What matters most is that the tunnel exits *outside* Myanmar with encrypted, hard-to-block traffic.

## Why LocalToNet instead of dedicated regional VPS exit nodes

The alternative (a real WireGuard VPS in each advertised region) gives a genuinely correct exit IP per location and no third party in the traffic path, at the cost of renting one VPS per region (~$4-6/mo each). We chose LocalToNet + 1-2 central servers instead, consciously accepting:

- **ToS risk.** LocalToNet's stated positioning is dev-tunnel/webhook/game-server exposure, not reselling VPN service to end users. Their actual ToS/AUP has not yet been read line-by-line — do that before scaling up usage. If violated, the account (and therefore every location at once) can be suspended without notice.
- **Metadata exposure.** LocalToNet's relay sees connection metadata (source IP, timing, volume) for every session, even though WireGuard's payload itself stays encrypted.
- **Single point of failure.** All locations run through the same 1-2 central servers — an outage or block there takes every location down at once, unlike independent regional nodes.
- **Cosmetic-only location selection**, as described above.

This tradeoff was chosen for cost and simplicity. Revisit if/when the user base or revenue justifies real regional VPS nodes — the control-plane API (`backend/`) is designed so that swapping "location → LocalToNet tunnel" for "location → dedicated VPS" later is a config change, not a rewrite (see `docs/BACKEND.md`).

## Censorship resistance (near-term concern, not yet solved)

Myanmar's network filtering can detect and block VPN protocols via deep packet inspection — plain WireGuard's handshake is fingerprintable and blockable outright. This is not addressed by the current design. Worth evaluating later: DPI-resistant WireGuard variants (e.g. AmneziaWG) or an obfuscation layer in front of the tunnel. Tracked in `docs/OPEN_QUESTIONS.md`.

## Connection flow (user's perspective)

1. App opens, user authenticates (see `docs/OPEN_QUESTIONS.md` for auth strategy — not yet decided).
2. App requests the location list from the control plane.
3. User taps a location. App calls the control plane: "give me a config for Singapore."
4. Control plane generates/reuses a WireGuard keypair for this device, registers the public key on the central WireGuard server, returns a config pointed at that location's LocalToNet relay address.
5. App hands the config to WireGuard's official Android tunnel library. Tunnel comes up through the relay to the central server, which decrypts and forwards to the real internet.

## Open architectural questions

See `docs/OPEN_QUESTIONS.md` — notably: LocalToNet ToS review, DB choice, auth strategy, which locations to launch with, DPI resistance.
