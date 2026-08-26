# Mobile app

## Approach: native Android now, thin client, native iOS later

Android app: **Kotlin + Jetpack Compose**, using WireGuard's official Android library (`com.wireguard.android`) for the actual tunnel.

"Thin client" means: as little business logic as possible lives in the app itself. Auth, the location list, tier/ads status, and WireGuard config generation all come from the backend API (`backend/`). The app's job is mostly:

1. Call the backend API (auth, get location list, request a config for a location).
2. Show a simple, friendly location picker + connect button.
3. Hand the returned WireGuard config to the WireGuard Android library and reflect connection state in the UI.
4. Show ads per whatever ad SDK is chosen (details pending — see `docs/OPEN_QUESTIONS.md`), respecting split-tunneling so the VPN tunnel doesn't interfere with ad SDK network calls if that turns out to be necessary.

## Why this makes iOS later easy

Because the logic (auth, tiers, location list, config generation) lives in the backend, the future iOS app doesn't need to reimplement any of that — it calls the same REST API and hands the config to WireGuard's official iOS library. What gets rebuilt for iOS is UI + platform tunnel plumbing, not business logic. This is why we did **not** need Flutter or Kotlin Multiplatform to hit the "iOS later without much rework" goal — the reuse happens at the API layer instead of in shared mobile code.

## UI/UX priorities

User asked explicitly for the app to be **really user-friendly and easy to use**. Concretely, that means for the core flow:

- One-tap connect: open app → big connect button, sensible default location pre-selected → tap → connected. No mandatory setup friction before the first connection.
- Location picker should be simple (flag/name + a plain-language relative-speed indicator), not a technical list of IPs or protocol jargon.
- Be transparent that location = connection point, not "you'll appear to be in this country" (see `docs/ARCHITECTURE.md`) — phrase this in plain, reassuring language, not a technical disclaimer that alarms users.
- Clear, unambiguous connected/disconnected state (color + icon + text, not just one of those).
- Ads should not block or delay the core connect flow — exact placement depends on the ad details you'll provide (`docs/OPEN_QUESTIONS.md`).

Detailed screen-by-screen UX will be designed once ad details and app name/branding are available, so mockups don't need to be redone.

## Current state

`android/` is a placeholder — no Android Studio project generated yet. Waiting on app name / package ID before scaffolding the actual Gradle project (see `docs/OPEN_QUESTIONS.md`).
