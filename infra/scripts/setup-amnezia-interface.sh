#!/usr/bin/env bash
# Brings up awg0 — an AmneziaWG userspace interface, parallel to and
# independent from the existing kernel wg0 (see setup-central-server.sh):
# different port, different subnet, no shared state. Existing wg0 traffic/
# users are completely unaffected by running this.
#
# Prereqs: infra/scripts/build-amneziawg.sh has been run and
# infra/build/{amneziawg-go,amnezia-setup} copied to this host (e.g. into
# /tmp), and AWG_* obfuscation params are set in the environment (see
# infra/scripts/generate-amnezia-params.sh) — required, not optional, since
# a missing set here would mean client configs (built by backend/ from the
# same AWG_* vars — see backend/.env) don't match what this interface is
# actually configured with, and no client would ever complete a handshake.
#
# Run as root on the central VPS:
#   AWG_JC=... AWG_JMIN=... ... ./setup-amnezia-interface.sh
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "Run as root." >&2
  exit 1
fi

: "${AWG_JC:?AWG_* obfuscation params must be set — see infra/scripts/generate-amnezia-params.sh}"

WG_PORT="${AWG_PORT:-51821}"
WG_SUBNET="${AWG_SUBNET:-10.67.0.0/16}"
WG_SERVER_IP="${AWG_SERVER_IP:-10.67.0.1}"
WG_IFACE="awg0"
BIN_DIR=/usr/local/bin
KEY_DIR=/etc/amneziawg
BUILD_SRC="${BUILD_SRC:-/tmp/amnezia-build}"

for f in amneziawg-go amnezia-setup; do
  if [[ ! -f "${BUILD_SRC}/${f}" ]]; then
    echo "missing ${BUILD_SRC}/${f} — run build-amneziawg.sh and copy infra/build/ here first" >&2
    exit 1
  fi
  install -m 755 "${BUILD_SRC}/${f}" "${BIN_DIR}/${f}"
done

mkdir -p "$KEY_DIR"
chmod 700 "$KEY_DIR"

echo "==> starting amneziawg-go (creates ${WG_IFACE} TUN + UAPI socket)"
cat > "/etc/systemd/system/amneziawg-${WG_IFACE}.service" <<EOF
[Unit]
Description=AmneziaWG userspace interface (${WG_IFACE})
After=network.target

[Service]
Type=forking
ExecStart=${BIN_DIR}/amneziawg-go ${WG_IFACE}
ExecStop=/usr/sbin/ip link del ${WG_IFACE}
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable "amneziawg-${WG_IFACE}"
systemctl restart "amneziawg-${WG_IFACE}"

# amneziawg-go backgrounds itself once the interface exists, but that can
# take a moment — poll for the UAPI socket rather than a fixed sleep.
for i in $(seq 1 20); do
  [[ -S "/var/run/amneziawg/${WG_IFACE}.sock" ]] && break
  sleep 0.5
done
if [[ ! -S "/var/run/amneziawg/${WG_IFACE}.sock" ]]; then
  echo "amneziawg-go did not create its UAPI socket — check: journalctl -u amneziawg-${WG_IFACE}" >&2
  exit 1
fi

echo "==> assigning address + bringing up ${WG_IFACE}"
ip addr replace "${WG_SERVER_IP}/16" dev "$WG_IFACE"
ip link set "$WG_IFACE" up

echo "==> configuring private key, listen port, and AmneziaWG obfuscation params"
SERVER_PUBLIC_KEY=$("${BIN_DIR}/amnezia-setup" "$WG_IFACE" "$WG_PORT" "$KEY_DIR")

echo "==> NAT/forwarding rules (idempotent: -C check before -A insert)"
EGRESS_IFACE=$(ip route show default | awk '{print $5; exit}')
iptables -t nat -C POSTROUTING -s "$WG_SUBNET" -o "$EGRESS_IFACE" -j MASQUERADE 2>/dev/null || \
  iptables -t nat -A POSTROUTING -s "$WG_SUBNET" -o "$EGRESS_IFACE" -j MASQUERADE
iptables -C FORWARD -i "$WG_IFACE" -j ACCEPT 2>/dev/null || iptables -A FORWARD -i "$WG_IFACE" -j ACCEPT
iptables -C FORWARD -o "$WG_IFACE" -j ACCEPT 2>/dev/null || iptables -A FORWARD -o "$WG_IFACE" -j ACCEPT

if ! grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.d/99-wireguard.conf 2>/dev/null; then
  echo "net.ipv4.ip_forward=1" > /etc/sysctl.d/99-wireguard.conf
  sysctl -p /etc/sysctl.d/99-wireguard.conf
fi

echo "=================================================================="
echo "AmneziaWG is up on ${WG_IFACE}, listening on UDP ${WG_PORT}."
echo "Server public key: ${SERVER_PUBLIC_KEY}"
echo ""
echo "Next steps:"
echo "  1. Point a NEW LocalToNet UDP tunnel at this host:${WG_PORT} (see"
echo "     infra/LOCALTONET_SETUP.md — same AuthToken, new tunnel/local port)."
echo "  2. Add a new location entry to backend/internal/servers/locations.json"
echo "     using that relay address, plus WG_INTERFACE=${WG_IFACE} and the"
echo "     same AWG_* vars used above in backend/.env — see"
echo "     backend/internal/servers/amnezia.go."
echo "  3. Mount /var/run/amneziawg into the api container (see"
echo "     backend/docker-compose.yml) so the backend can reach this"
echo "     interface's UAPI socket, then redeploy."
echo "=================================================================="
