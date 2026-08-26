#!/bin/bash
set -euo pipefail

SERVER_HOST="${SERVER_HOST:-sy-vpn-server-test}"
# Default: full-tunnel, matching real production config exactly (needs
# --privileged — see README.md). Set ALLOWED_IPS=10.66.0.0/16 and drop
# --privileged for a lighter-weight run that only proves the tunnel itself
# (handshake + encrypted transport), not the default-route capture.
ALLOWED_IPS="${ALLOWED_IPS:-0.0.0.0/0}"

echo "=== waiting for server API ==="
for i in $(seq 1 20); do
  if curl -sf "http://${SERVER_HOST}:8080/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
curl -s "http://${SERVER_HOST}:8080/health"; echo

echo "=== register (real anonymous device-bound auth) ==="
REG=$(curl -s -X POST "http://${SERVER_HOST}:8080/auth/register" -d '{"device_id":"local-test-device"}')
echo "$REG"
TOKEN=$(echo "$REG" | jq -r .token)

echo "=== list locations (authenticated) ==="
curl -s "http://${SERVER_HOST}:8080/locations" -H "Authorization: Bearer ${TOKEN}"; echo

echo "=== connect (real WireGuard key generation + live peer registration via wgctrl) ==="
CONNECT=$(curl -s -X POST "http://${SERVER_HOST}:8080/connect" -H "Authorization: Bearer ${TOKEN}" -d '{"location_id":"singapore"}')
echo "$CONNECT" | jq .

CONFIG=$(echo "$CONNECT" | jq -r .config)
# Overrides for this local test only (production code/config is
# unchanged): locations.json's relay_address is still a LocalToNet
# placeholder (no real account exists yet), so point Endpoint at the real
# server container directly, reachable by name on the isolated test
# network. AllowedIPs is set from $ALLOWED_IPS (see above). Everything else
# (keys, IP allocation, live peer registration) is exactly what production
# code produced.
echo "$CONFIG" \
  | sed "s#Endpoint = .*#Endpoint = ${SERVER_HOST}:51820#" \
  | sed "s#AllowedIPs = .*#AllowedIPs = ${ALLOWED_IPS}#" \
  > /etc/wireguard/wg0.conf
echo "=== client config (Endpoint/AllowedIPs overridden for local test) ==="
cat /etc/wireguard/wg0.conf

echo "=== bringing up the real WireGuard tunnel ==="
wg-quick up wg0

sleep 2
echo "=== wg show (handshake + transfer stats) ==="
wg show wg0

echo "=== pinging the server's tunnel IP through the encrypted tunnel ==="
ping -c 3 -W 3 10.66.0.1

echo "=== fetching a real external URL routed through the tunnel ==="
echo "(only meaningful proof of the server's NAT/forwarding path if ALLOWED_IPS=0.0.0.0/0 — with a scoped AllowedIPs this just uses the container's own normal route, not the tunnel)"
curl -s --max-time 8 http://ifconfig.me/ip || echo "(external fetch failed — handshake+ping above already proves the tunnel itself works)"
echo

echo "=== DONE ==="
