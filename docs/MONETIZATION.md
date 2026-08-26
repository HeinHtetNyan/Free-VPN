# Monetization

## Current decision

Free to use, supported by ads. An optional paid tier ("buy to remove ads" or similar) may be added later — not designed yet, and not blocking anything now.

## Ad network: Adsterra

Decided. Important technical fact this changes: **Adsterra has no native Android SDK** — it's a web/webview ad network (Banner, Native, Interstitial, Social Bar, Popunder, In-Page Push, Smartlink), integrated via WebView + ad tag/JS rather than a purpose-built mobile SDK like AdMob.

Practical consequences:

- **Format choice matters for Play Store survival.** Popunder and Social Bar formats open new contexts or overlay content in ways that commonly trigger Play Store policy rejections (ads disguised as/interfering with system UI). Stick to **Banner, Native, and Interstitial** rendered inside a contained WebView. Avoid Popunder/Social Bar in the shipped app.
- **Integration is a WebView loading Adsterra's ad tag**, not a native `AdView`-style widget. Plan the Android UI accordingly — a contained WebView component sized/placed like a normal banner/interstitial, not a full navigation-away experience.
- **Split tunneling may be needed.** The WebView's requests to Adsterra's ad-serving domains will go through the VPN tunnel by default. If that causes ad load failures/slowness (tunnel down, or ad domains blocked upstream of the tunnel), exclude the ad WebView's traffic from the tunnel via Android's `VpnService` allow/disallow-list. Decide once real-world testing shows whether this is actually a problem — don't build it preemptively.

## How this shapes the backend now

`internal/users` models a simple tier flag on the user record (e.g. `free` today, room for `premium` later) rather than building out real billing/subscription plumbing before it's needed. When a paid tier is designed, this becomes: add a payment provider integration (likely Google Play Billing for Android) that flips this flag — not a restructure.

## Still open

- **Ad placement** — e.g. interstitial on connect/disconnect, banner during connected state, rewarded ad for temporary premium-location access — needs to not add friction to the core "tap to connect" flow (see `docs/MOBILE.md` UX priorities).
- **Adsterra publisher account / zone IDs** — needed before real integration, not yet provided.

## Open question

Whether ads should be blocked while the user is actively connected (some VPN products avoid interstitials mid-session to not feel like they're interrupting a "private browsing" experience) vs. shown at connect/disconnect transition points only. Revisit with ad details.
