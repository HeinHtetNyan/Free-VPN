# Privacy Policy — DRAFT, not published anywhere yet

Written against what the app **actually does** as of 2026-08-26 (see `docs/DECISIONS.md`), so it doesn't drift from reality the way a generic template would. **Not legal advice, not reviewed by a lawyer, and not yet published in-app or in the Play Store listing** — both are required before submission (`docs/PLAY_STORE_COMPLIANCE.md`). Remaining placeholders (`[CONTACT EMAIL]`, `[DATE]`) need real values before publishing.

Update this doc whenever the underlying behavior changes (new ad network, real accounts added, analytics added, etc.) — it must always describe what the app actually does, not what it did when first written.

---

## Privacy Policy for SY VPN

*Last updated: [DATE]*

### What this app is

SY VPN is a VPN app that routes your internet traffic through an encrypted tunnel (WireGuard) to help you reach the open internet.

### Information we collect

**Device identity.** On first launch, the app generates a random identifier on your device and sends it to our server to create an account. We do not ask for your name, email, or phone number, and no real-world identity is tied to this account.

**Connection metadata visible to our infrastructure providers.** We use a third-party relay service to offer multiple connection locations without requiring a server in every country. This means the relay provider can see connection metadata — your IP address, connection timing, and data volume — for sessions passing through it, even though the contents of your traffic remain encrypted end-to-end via WireGuard and unreadable to them. We do not believe this provider inspects or logs traffic content, but we have not independently audited this, and we do not control their infrastructure.

**What we do not collect.** We do not log which websites or services you visit, the contents of your traffic, or your browsing history. [Confirm this stays true before publishing — this is a strong claim; see `docs/BACKEND.md` for what the server code actually logs today, and update this section if that changes.]

**Advertising data.** This app shows ads via Adsterra. Adsterra may collect information such as your device's advertising identifier, IP address, and general location to serve and measure ads. This happens independently of the VPN tunnel's connection metadata above. See Adsterra's own privacy policy for details on their data practices: [LINK TO ADSTERRA PRIVACY POLICY].

### How we use this information

- Device identity: to associate your app with your connection configuration and (if/when introduced) any paid tier.
- Connection metadata visible to our relay provider: not used by us directly; disclosed here because it exists and you should know about it.
- Advertising data: to display ads that keep the app free, and (via Adsterra) to measure ad performance.

### Data retention

[Not yet decided — fill in once the backend has an actual retention/deletion policy rather than indefinite storage by default. Currently: user accounts persist until manually deleted; no automatic expiry implemented yet.]

### Your choices

- You can stop using the app and its associated device identity at any time; since no real-world identity is tied to the account, there's nothing further to delete on our end beyond the account record itself. [Confirm/add an in-app "delete my data" action before publishing, if one gets built.]

### Third parties

- **[LocalToNet]** — connection relay. See "Connection metadata" above.
- **Adsterra** — advertising.
- We do not sell your data to anyone.

### Security

Your traffic is encrypted end-to-end using WireGuard between your device and our server. [Add any additional security claims only once true and verified — see `docs/PLAY_STORE_COMPLIANCE.md` "Don't overclaim privacy/security."]

### Changes to this policy

We may update this policy as the app changes. [Add a real changelog/versioning approach before publishing.]

### Contact

[CONTACT EMAIL]

---

## Checklist before this can actually be published

- [ ] Replace all bracketed placeholders with real values.
- [ ] Confirm the "what we do not collect" claims are still true against the actual backend code at publish time.
- [ ] Add the real LocalToNet privacy policy link once an account exists, and re-read their policy to make sure this doc's characterization of what they can see is accurate.
- [ ] Add the real Adsterra privacy policy link.
- [ ] Have someone (ideally with legal knowledge) actually review this before it goes live — this draft is a starting point, not a finished legal document.
- [ ] Publish somewhere with a stable URL, then link it in-app and in the Play Store listing (`docs/PLAY_STORE_COMPLIANCE.md`).
