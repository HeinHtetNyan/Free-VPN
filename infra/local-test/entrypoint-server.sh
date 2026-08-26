#!/bin/bash
set -euo pipefail

WG_PORT=51820
WG_SUBNET=10.66.0.0/16

ip link add wg0 type wireguard
wg genkey | tee /tmp/server_private.key | wg pubkey > /tmp/server_public.key
ip addr add 10.66.0.1/16 dev wg0
wg set wg0 listen-port "$WG_PORT" private-key /tmp/server_private.key
ip link set wg0 up


EGRESS_IFACE=$(ip route show default | awk '{print $5; exit}')
iptables -t nat -A POSTROUTING -s "$WG_SUBNET" -o "$EGRESS_IFACE" -j MASQUERADE
iptables -A FORWARD -i wg0 -j ACCEPT
iptables -A FORWARD -o wg0 -j ACCEPT

export WG_INTERFACE=wg0
export DB_PATH=/tmp/test.db
export SERVER_PUBLIC_KEY
SERVER_PUBLIC_KEY=$(cat /tmp/server_public.key)
export PORT=8080

echo "=================================================================="
echo "Local test WireGuard server up. Public key: ${SERVER_PUBLIC_KEY}"
echo "=================================================================="

exec /usr/local/bin/sy-vpn-server
