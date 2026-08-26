# Privacy Policy — published

Written against what the app **actually does** as of 2026-08-27 (see `docs/DECISIONS.md`). Served live from the backend at `GET /privacy` (`backend/internal/api/privacy.go`) — that route is the source of truth; this file is a plain-text mirror for reference/editing. **Not legal advice, not reviewed by a lawyer.**

Update this doc *and* `backend/internal/api/privacy.go` together whenever the underlying behavior changes (new ad network, real accounts added, analytics added, etc.) — it must always describe what the app actually does, not what it did when first written.

Live URL (once pushed + deployed): **https://sy-api.heinh.dev/privacy**

---

## Privacy Policy for SY VPN

*Effective: August 27, 2026*

SY VPN ("the app") is an Android VPN app built on WireGuard. This page explains what information the app collects, why, and how it's used. There is no user account, no sign-up, and no name, email, or phone number collected to use the app.

### Information We Collect

- **Anonymous device ID.** On first launch, the app generates a random identifier stored only on your device. It is not linked to your name, email, or phone number, and is used solely to issue a connection token and attribute aggregate usage (see below) to a device rather than a person.
- **Connection & usage data.** While you use the VPN, we record which relay location you connected to and how much data was transferred, associated with your anonymous device ID — used to monitor server capacity and reliability. We do **not** log the websites you visit, your DNS queries, or the contents of your traffic.
- **Issue reports (optional).** If you tap "Report an issue," we receive the message you type, the network/carrier name you enter, and basic device info (device model, Android version, app version). This is only sent when you choose to submit a report, and is used to diagnose connectivity problems — for example, identifying which mobile carriers block or throttle the VPN.

### Advertising

The app shows ads served by Adsterra (adsterra.com). Adsterra may collect device or advertising identifiers and use cookies to serve and measure ads, under its own privacy policy. We don't control or receive this data ourselves beyond standard ad-performance reporting.

### How We Use Information

- To operate and maintain the VPN service (issuing connections, routing traffic).
- To monitor server load and capacity across relay locations.
- To diagnose connectivity issues reported by users, including ISP-level blocking.

### Data Sharing

We do not sell your data. Information is shared only with the infrastructure providers necessary to run the service (hosting and network relay providers) and with Adsterra for ad serving, as described above.

### Data Retention

Your anonymous device ID and connection token are stored locally on your device until you uninstall the app. Aggregate usage and report data on our servers is retained only as long as needed for service operation and diagnostics.

### Security

VPN traffic is encrypted using the WireGuard protocol. A fresh cryptographic key pair is generated for every connection.

### Children's Privacy

SY VPN is not directed at children under 13, and we do not knowingly collect information from children.

### Your Choices

You can stop using the service and delete all local app data at any time by uninstalling the app. To request deletion of server-side usage/report records tied to your anonymous ID, email the contact below.

### Changes to This Policy

We may update this policy from time to time. Changes will be posted on this page with an updated effective date.

### Contact

hhn@heinh.dev

---

## Remaining before this is fully airtight

- [ ] Link this page in-app (not just in the Play Store listing) — currently only linked from the store listing draft.
- [ ] Once a real LocalToNet account/relay is in play for a *second* location, re-check whether it changes what "Connection & usage data" should disclose (current single relay's visibility was covered under the general infra-provider language).
- [ ] Once the real Adsterra ad tag replaces the placeholder in `AdsterraBannerAd.kt`, confirm whether it reads the Android Advertising ID — if so, add that specifically to this page and to the Data Safety form (`PLAY_STORE_LISTING.md`).
- [ ] Legal review before treating this as final.
