# Glossary

Terms used throughout `docs/`, in the order you'll run into them.

**Control plane**
The central backend service(s) that don't carry user VPN traffic. Handles authentication, user accounts, the list of available server locations, and telling exit nodes which users are allowed to connect. Lives in `backend/`.

**Central server**
One of the 1-2 VPS instances that run both the control plane and the actual WireGuard VPN termination. Unlike a per-region exit node, all locations funnel back to the same central server(s) — see `docs/ARCHITECTURE.md` for why, and the tradeoffs of that choice.

**LocalToNet relay**
A LocalToNet-operated server in a given region (e.g. Singapore, Tokyo) that forwards a user's WireGuard (UDP) connection back to our central server. It's what makes a "location" selectable in the app without us renting a VPS in that region — but it does not change where traffic exits to the internet (that's always the central server's location). See `docs/ARCHITECTURE.md`.

**WireGuard**
The VPN protocol/software we're building on. Fast, modern, small codebase, has official client libraries for Android and iOS. Each connection is between a "peer" (the user's phone) and a WireGuard server (an exit node).

**Peer**
One WireGuard identity — a public/private keypair. Each user's device is a peer. "Provisioning a peer" means generating a keypair and registering the public half on an exit node so that device is allowed to connect.

**Tunnel**
The encrypted connection between the user's phone and an exit node, once WireGuard is set up. All the user's internet traffic flows through it while the VPN is on.

**Split tunneling**
Excluding specific apps or traffic from the VPN tunnel — e.g. letting the VPN app's own ad-SDK network calls go over the normal connection instead of through the tunnel, if needed. Android's `VpnService` API supports this natively (`addDisallowedApplication` / `addAllowedApplication`).

**Thin client**
Our approach for the mobile apps: as much logic as possible (auth, server list, subscription/tier status) lives in the backend API. The Android/iOS apps mostly just call that API and hand a config to the platform's WireGuard library. Keeps the future iOS app small to build, since it's not re-implementing business logic.

**DPI (Deep Packet Inspection)**
A censorship technique where network traffic is inspected for protocol fingerprints (not just blocked by IP/port) to detect and block things like VPN handshakes. Relevant because Myanmar's network filtering is known to do this — plain WireGuard can be fingerprinted and blocked. See `docs/ARCHITECTURE.md` "Censorship resistance."

**Tier**
A user's access level — currently just free-with-ads. A paid tier may be added later (not yet decided how). The backend models this as a flag on the user record so adding a real tier later doesn't require restructuring.
