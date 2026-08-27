# Play Store submission kit — SY VPN

Everything needed to fill out Google Play Console's store listing and Data Safety form. Drafted from what the app's code actually does as of 2026-08-27, not boilerplate — cross-check against `docs/PLAY_STORE_COMPLIANCE.md` before submitting.

Package name: `com.syvpn.app`

## 1. Store listing copy

**App name** (7 chars)
```
SY VPN
```

**Short description** (78 / 80 chars)
```
Fast, no-signup WireGuard VPN with live server latency and no traffic logs.
```

**Full description** (~980 / 4000 chars)
```
SY VPN — simple, fast, and private.

Connect in one tap. There's no account, no email, and no sign-up — SY VPN generates an anonymous ID the moment you open the app, so there's nothing tying your VPN use to your identity.

WHY SY VPN
• One-tap connect — no account needed
• See real, live latency before you connect, so you always know your connection quality
• No traffic logs — we never record the websites you visit, your DNS queries, or your connection contents
• Built on WireGuard — the fast, modern, open-source VPN protocol
• See how many people are connected right now, live in the app

REPORT WHAT YOU SEE
Some mobile carriers restrict VPN traffic. If SY VPN isn't working well on your network, the in-app "Report an issue" button lets you tell us your carrier and what happened in a few taps — so we can find and fix it.

PRIVACY BY DESIGN
SY VPN doesn't ask for your name, email, or phone number. Your anonymous ID lives only on your device until you uninstall the app. Full privacy policy: sy-api.heinh.dev/privacy

SY VPN is in early access — more server locations are being added over time. Right now, SY VPN connects you through our Singapore relay.
```

**Category**: Tools (fallback: Communication)

> Note: "early access / more locations coming" is deliberate — only Singapore is live right now (`docs/DECISIONS.md`). Drop that line once a second location ships, per the Deceptive Behavior policy note in `PLAY_STORE_COMPLIANCE.md`.

**Privacy policy URL**
```
https://sy-api.heinh.dev/privacy
```

## 2. Data Safety form

Google can suspend a listing over an inaccurate Data Safety declaration — treat this as a strong first draft, not a final answer. Walk through Play Console's own category definitions before submitting.

| Category | Collected? | Notes |
|---|---|---|
| Location | No | No location permission anywhere in the app. |
| Personal info | No | No name, email, phone number, or address — ever. |
| Financial info | No | No payments in-app. |
| Messages | No | Not applicable — no SMS/email/chat access. |
| Photos, videos, audio, files | No | — |
| App activity → other user-generated content | **Yes** | The optional issue-report message. Purpose: App functionality. Optional, user-initiated, not shared with third parties. |
| Web browsing history | No | The whole point of a no-logs VPN — traffic contents/destinations aren't recorded. |
| App info & performance → diagnostics | No | No crash-reporting SDK integrated yet. |
| Device or other IDs | **Yes** | App-generated random UUID (not IMEI/Android ID), stored on-device, used to issue a connection token and attribute usage. **Also**: the Adsterra banner (loaded via WebView, not their native SDK — see below), for advertising purposes, shared with Adsterra. |

**Encryption**: Data is encrypted in transit — Yes (HTTPS/TLS to the backend, WireGuard encryption for the VPN tunnel itself).

### Two judgment calls before you submit

1. **Anonymous ID classification.** It's app-generated and resets on uninstall — arguably not a "device ID" in Play's strict sense (that category is meant for hardware identifiers like IMEI). Declaring it anyway is the safer, more conservative choice.
2. **Adsterra + Advertising ID — resolved 2026-08-27.** The real ad tag is now wired in (`AdsterraBannerAd.kt`, zone ID from `local.properties`), loaded via WebView + JS, not Adsterra's native SDK — so this app never calls Android's `AdvertisingIdClient` API directly. However, Adsterra's own privacy policy (`adsterra.com`) states they collect advertising identifiers (AAID/IDFA) as part of serving ads generally. Per the same conservative principle as the anonymous-ID call above: declared as collected/shared with Adsterra for advertising purposes in the table above, even though this app's own code doesn't request it directly — the safer assumption given Adsterra's script runs inside the WebView with its own tracking mechanisms outside this app's control.

### Data deletion

There's no account, so uninstalling removes the on-device ID. Google's newer Data Safety requirements increasingly expect an explicit deletion path even for anonymous data — state in the form that users can request deletion of their server-side usage/report records by emailing **hhn@heinh.dev**.

## 3. Asset checklist

- [x] **App icon (512×512)** — already have it, the real SY VPN logo, same one used for the launcher icon.
- [x] **Phone screenshots (2+ needed)** — captured live from the test device: the connect screen (idle + latency shown) and the "Report an issue" dialog. Usable as-is; can re-capture from a release build instead of debug if preferred.
- [x] **Feature graphic (1024×500)** — `docs/store-assets/feature-graphic-1024x500.png`, built from the real logo and "night signal" palette.
- [x] **Signed release build (AAB)** — `android/app/build/outputs/bundle/release/sy-vpn-release.aab`, signed with `android/release.keystore` (verified via `keytool -printcert`, not the debug keystore). Rebuild after any code change before uploading — this one's from 2026-08-27.
