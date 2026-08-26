#!/usr/bin/env bash
# Installs and configures WireGuard on a central VPS as a complete, running
# wg0 interface — not just package install. See ../../docs/ARCHITECTURE.md.
#
# Run as root on a fresh Debian/Ubuntu VPS:
#   WG_PORT=51820 ./setup-central-server.sh
#
# After this script: follow ../LOCALTONET_SETUP.md to point each location's
# LocalToNet relay at this server's WG_PORT, and put the printed server
# public key into backend's placeholderServerPublicKey
# (backend/internal/api/handlers.go) — see ../../docs/OPEN_QUESTIONS.md.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "Run as root." >&2
  exit 1
fi

WG_PORT="${WG_PORT:-51820}"
WG_SUBNET="${WG_SUBNET:-10.66.0.0/16}"
WG_SERVER_IP="${WG_SERVER_IP:-10.66.0.1}"
WG_IFACE="wg0"

apt-get update
apt-get install -y wireguard wireguard-tools iptables

umask 077
mkdir -p /etc/wireguard
cd /etc/wireguard

if [[ ! -f server_private.key ]]; then
  wg genkey | tee server_private.key | wg pubkey > server_public.key
fi
SERVER_PRIVATE_KEY=$(cat server_private.key)
SERVER_PUBLIC_KEY=$(cat server_public.key)

# The egress interface NAT rules below need to target — assumes a single
# default route, true for essentially every VPS.
EGRESS_IFACE=$(ip route show default | awk '{print $5; exit}')

cat > "/etc/wireguard/${WG_IFACE}.conf" <<EOF
[Interface]
Address = ${WG_SERVER_IP}/16
ListenPort = ${WG_PORT}
PrivateKey = ${SERVER_PRIVATE_KEY}
# Peers are added at runtime by the control plane (backend/internal/servers
# RegisterPeer via wgctrl) — not written into this file by this script.

PostUp = iptables -t nat -A POSTROUTING -s ${WG_SUBNET} -o ${EGRESS_IFACE} -j MASQUERADE
PostUp = iptables -A FORWARD -i ${WG_IFACE} -j ACCEPT
PostUp = iptables -A FORWARD -o ${WG_IFACE} -j ACCEPT
PostDown = iptables -t nat -D POSTROUTING -s ${WG_SUBNET} -o ${EGRESS_IFACE} -j MASQUERADE
PostDown = iptables -D FORWARD -i ${WG_IFACE} -j ACCEPT
PostDown = iptables -D FORWARD -o ${WG_IFACE} -j ACCEPT
EOF
chmod 600 "/etc/wireguard/${WG_IFACE}.conf"

# Persist IPv4 forwarding across reboots.
if ! grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.d/99-wireguard.conf 2>/dev/null; then
  echo "net.ipv4.ip_forward=1" > /etc/sysctl.d/99-wireguard.conf
  sysctl -p /etc/sysctl.d/99-wireguard.conf
fi

systemctl enable "wg-quick@${WG_IFACE}"
systemctl restart "wg-quick@${WG_IFACE}"

echo "=================================================================="
echo "WireGuard is up on ${WG_IFACE}, listening on UDP ${WG_PORT}."
echo "Server public key: ${SERVER_PUBLIC_KEY}"
echo ""
echo "Next steps (see ../../docs/OPEN_QUESTIONS.md):"
echo "  1. Point each location's LocalToNet UDP tunnel at this host:${WG_PORT}."
echo "  2. Put the public key above into backend/internal/api/handlers.go"
echo "     (placeholderServerPublicKey) and each location's relay_address"
echo "     into backend/internal/servers/locations.json."
echo "  3. Set WG_INTERFACE=${WG_IFACE} when running backend/ on this host"
echo "     (it's already the default)."
echo "=================================================================="
