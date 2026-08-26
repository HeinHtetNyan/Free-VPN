# Play Store compliance

VPN apps get extra Play Store scrutiny (their "Device and Network Abuse" policy applies specifically to VPN apps), and a webview-based ad network adds more risk surface on top. This doc consolidates the checklist items that are otherwise scattered across `MOBILE.md`, `MONETIZATION.md`, and `ARCHITECTURE.md`, so compliance is a deliberate design constraint, not an afterthought before submission.

This is a working checklist, not legal review — read Google Play's actual current "Device and Network Abuse" and "Ads" policy pages before submitting; policy specifics change.

## VPN-specific requirements

- Use Android's `VpnService` API properly (already the plan — no shortcuts through a proxy hack).
- **Accurate, published Privacy Policy**, linked in-app and in the Play Store listing. For this app that must disclose: what LocalToNet (a third-party relay) can see (connection metadata — see `ARCHITECTURE.md`), what Adsterra collects for ad serving, and whatever auth data is collected once that's decided (`OPEN_QUESTIONS.md`). This is a hard requirement for VPN apps, not optional boilerplate.
- **Don't overclaim privacy/security.** No "100% anonymous," "no one can see your traffic," "unhackable" style copy — given LocalToNet sees connection metadata, an absolute-anonymity claim would be both false and a Deceptive Behavior policy violation. Review marketing copy against this once written.
- App description must clearly and accurately describe VPN functionality.

## Deceptive Behavior policy — ties directly to how "location selection" is designed

Since exit IP doesn't actually change per selected location (see `ARCHITECTURE.md`), presenting "Thailand server" as if it changes the user's apparent country, without disclosure, is exactly the kind of misleading claim the Deceptive Behavior policy targets. The earlier decision to be transparent in the UI (`DECISIONS.md`, 2026-08-26) isn't just a UX nicety — treat it as a compliance requirement: label locations as "connection points" / "fastest server," never "browse as if you're in this country."

## Ads (Adsterra) policy risk

Already decided in `MONETIZATION.md`: Banner/Native/Interstitial only, no Popunder/Social Bar — those formats are the main Play policy trigger. Hold to these as well:

- Interstitials need an obvious, functioning close button — never a forced click-through.
- No ads that auto-redirect the user (to Play Store, another app, or a browser) without a deliberate tap.
- Ads must not overlap/obscure app navigation (e.g. sit on top of the connect button) or mimic system UI/notifications.
- No ads placed in the system notification shade.

## Data Safety form

Play Console requires a "Data safety" declaration that must match real app behavior. Once auth strategy + Adsterra integration are final, this needs to accurately state: what's collected, whether it's shared with third parties (LocalToNet's metadata visibility and Adsterra's ad-serving both count as "shared with third parties"), and whether it's encrypted in transit. A mismatch between this form and actual behavior is itself a policy violation, independent of whether the underlying behavior is fine on its own.

## Open items

- [x] Write the actual Privacy Policy — `docs/PRIVACY_POLICY_DRAFT.md`, live at `GET /privacy` once pushed/deployed.
- [x] Draft app store listing copy — `docs/PLAY_STORE_LISTING.md`.
- [x] Draft the Data Safety form — `docs/PLAY_STORE_LISTING.md`, flagged for two judgment calls (anonymous ID classification, Adsterra ad-ID) once the real ad tag is wired in.
- [ ] Get a real Adsterra zone ID wired into `AdsterraBannerAd.kt` / `local.properties`, then re-check the Data Safety form's advertising-ID answer against Adsterra's actual behavior.
- [ ] Build a signed release AAB (everything tested so far is a debug build).
- [ ] Feature graphic (1024×500) — optional for publishing, not yet made.
- [ ] Re-check this list against Google's current published VPN/Ads policy text before first submission — don't rely solely on this doc at submission time.
