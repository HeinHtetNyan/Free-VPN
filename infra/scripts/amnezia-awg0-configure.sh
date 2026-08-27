#!/usr/bin/env bash
# Finishes bringing up awg0 after amneziawg-awg0.service (which only starts
# the raw amneziawg-go process) — mirrors the "assign address" / "configure
# private key, listen port, obfuscation params" / "NAT rules" steps from
# infra/scripts/setup-amnezia-interface.sh in the sy-vpn-backend repo, so
# they survive a reboot instead of needing to be re-run by hand. Reuses the
# existing keys in /etc/amneziawg (does not regenerate — that would break
# every already-issued client config). Idempotent: safe to run on every boot.
set -euo pipefail

WG_PORT="${AWG_PORT:-51821}"
WG_SUBNET="${AWG_SUBNET:-10.67.0.0/16}"
WG_SERVER_IP="${AWG_SERVER_IP:-10.67.0.1}"
WG_IFACE="awg0"
BIN_DIR=/usr/local/bin
KEY_DIR=/etc/amneziawg
ENV_FILE=/home/appbox/SY/backend/.env

# Not `source`d: values like AWG_I1=<r 30> are valid KEY=VALUE env-file
# syntax (that's how docker-compose's env_file reads it) but invalid bash
# (the unquoted `<` there parses as a redirect) — read and export literally
# instead of letting the shell reinterpret each value.
while IFS='=' read -r key value; do
  [[ -z "$key" || "$key" == \#* ]] && continue
  export "$key=$value"
done < "$ENV_FILE"

for i in $(seq 1 20); do
  [[ -S "/var/run/amneziawg/${WG_IFACE}.sock" ]] && break
  sleep 0.5
done
if [[ ! -S "/var/run/amneziawg/${WG_IFACE}.sock" ]]; then
  echo "amneziawg-go did not create its UAPI socket — check: journalctl -u amneziawg-${WG_IFACE}" >&2
  exit 1
fi

ip addr replace "${WG_SERVER_IP}/16" dev "$WG_IFACE"
ip link set "$WG_IFACE" up

"${BIN_DIR}/amnezia-setup" "$WG_IFACE" "$WG_PORT" "$KEY_DIR"

EGRESS_IFACE=$(ip route show default | awk '{print $5; exit}')
iptables -t nat -C POSTROUTING -s "$WG_SUBNET" -o "$EGRESS_IFACE" -j MASQUERADE 2>/dev/null || \
  iptables -t nat -A POSTROUTING -s "$WG_SUBNET" -o "$EGRESS_IFACE" -j MASQUERADE
iptables -C FORWARD -i "$WG_IFACE" -j ACCEPT 2>/dev/null || iptables -A FORWARD -i "$WG_IFACE" -j ACCEPT
iptables -C FORWARD -o "$WG_IFACE" -j ACCEPT 2>/dev/null || iptables -A FORWARD -o "$WG_IFACE" -j ACCEPT

if ! grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.d/99-wireguard.conf 2>/dev/null; then
  echo "net.ipv4.ip_forward=1" > /etc/sysctl.d/99-wireguard.conf
  sysctl -p /etc/sysctl.d/99-wireguard.conf
fi

echo "awg0 configured: ${WG_SERVER_IP}/16, listening on UDP ${WG_PORT}"
