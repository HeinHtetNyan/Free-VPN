package api

import "net/http"

// handlePrivacy serves the Play Store-required privacy policy at a stable,
// public URL (no auth) — see docs/PLAY_STORE_COMPLIANCE.md. Self-hosted here
// rather than a third-party doc host so the URL never breaks or requires a
// separate deploy.
func (s *Server) handlePrivacy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(privacyPolicyHTML))
}

const privacyPolicyHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>SY VPN — Privacy Policy</title>
<style>
  :root { color-scheme: light dark; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    max-width: 720px;
    margin: 0 auto;
    padding: 32px 20px 80px;
    line-height: 1.6;
    color: #1a1d24;
    background: #ffffff;
  }
  @media (prefers-color-scheme: dark) {
    body { color: #e4e7ee; background: #0b0e14; }
    a { color: #4fd1a5; }
    h2 { border-bottom-color: #262b36 !important; }
  }
  h1 { font-size: 1.6rem; margin-bottom: 4px; }
  .updated { color: #767b87; font-size: 0.85rem; margin-bottom: 32px; }
  h2 { font-size: 1.05rem; margin-top: 36px; border-bottom: 1px solid #e4e7ee; padding-bottom: 6px; }
  ul { padding-left: 20px; }
  li { margin-bottom: 6px; }
  a { color: #0f9d6f; }
  .contact { font-weight: 600; }
</style>
</head>
<body>
  <h1>SY VPN — Privacy Policy</h1>
  <p class="updated">Effective: August 27, 2026</p>

  <p>SY VPN ("the app") is an Android VPN app built on WireGuard. This page explains what
  information the app collects, why, and how it's used. There is no user account, no sign-up,
  and no name, email, or phone number collected to use the app.</p>

  <h2>Information We Collect</h2>
  <ul>
    <li><strong>Anonymous device ID.</strong> On first launch, the app generates a random
    identifier stored only on your device. It is not linked to your name, email, or phone
    number, and is used solely to issue a connection token and attribute aggregate usage
    (see below) to a device rather than a person.</li>
    <li><strong>Connection &amp; usage data.</strong> While you use the VPN, we record which
    relay location you connected to and how much data was transferred, associated with your
    anonymous device ID — used to monitor server capacity and reliability. We do <strong>not</strong>
    log the websites you visit, your DNS queries, or the contents of your traffic.</li>
    <li><strong>Issue reports (optional).</strong> If you tap "Report an issue," we receive
    the message you type, the network/carrier name you enter, and basic device info (device
    model, Android version, app version). This is only sent when you choose to submit a report,
    and is used to diagnose connectivity problems — for example, identifying which mobile
    carriers block or throttle the VPN.</li>
  </ul>

  <h2>Advertising</h2>
  <p>The app shows ads served by <a href="https://adsterra.com" target="_blank" rel="noopener">Adsterra</a>.
  Adsterra may collect device or advertising identifiers and use cookies to serve and measure
  ads, under its own privacy policy. We don't control or receive this data ourselves beyond
  standard ad-performance reporting.</p>

  <h2>How We Use Information</h2>
  <ul>
    <li>To operate and maintain the VPN service (issuing connections, routing traffic).</li>
    <li>To monitor server load and capacity across relay locations.</li>
    <li>To diagnose connectivity issues reported by users, including ISP-level blocking.</li>
  </ul>

  <h2>Data Sharing</h2>
  <p>We do not sell your data. Information is shared only with the infrastructure providers
  necessary to run the service (hosting and network relay providers) and with Adsterra for ad
  serving, as described above.</p>

  <h2>Data Retention</h2>
  <p>Your anonymous device ID and connection token are stored locally on your device until you
  uninstall the app. Aggregate usage and report data on our servers is retained only as long as
  needed for service operation and diagnostics.</p>

  <h2>Security</h2>
  <p>VPN traffic is encrypted using the WireGuard protocol. A fresh cryptographic key pair is
  generated for every connection.</p>

  <h2>Children's Privacy</h2>
  <p>SY VPN is not directed at children under 13, and we do not knowingly collect information
  from children.</p>

  <h2>Your Choices</h2>
  <p>You can stop using the service and delete all local app data at any time by uninstalling
  the app.</p>

  <h2>Changes to This Policy</h2>
  <p>We may update this policy from time to time. Changes will be posted on this page with an
  updated effective date.</p>

  <h2>Contact</h2>
  <p class="contact">hhn@heinh.dev</p>
</body>
</html>
`
