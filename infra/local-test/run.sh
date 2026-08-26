#!/usr/bin/env bash
# Proves the whole VPN mechanism (real key generation, live peer
# registration via wgctrl, WireGuard handshake, encrypted transport, and —
# with ALLOWED_IPS=0.0.0.0/0 (default) — server-side NAT/forwarding) using
# two disposable, isolated Docker containers. Touches NO host networking
# state — see README.md.
#
# Usage: ./run.sh [--keep] [--scoped]
#   --keep     don't tear down the server container/network afterward
#   --scoped   use AllowedIPs=10.66.0.0/16 and skip --privileged (lighter,
#              proves the tunnel itself but not the full default-route path)
set -euo pipefail
cd "$(dirname "$0")"

KEEP=false
SCOPED=false
for arg in "$@"; do
  case "$arg" in
    --keep) KEEP=true ;;
    --scoped) SCOPED=true ;;
  esac
done

echo "=== building backend binary ==="
( cd ../../backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../infra/local-test/sy-vpn-server ./cmd/server )

echo "=== building images ==="
docker build -f Dockerfile.server -t sy-vpn-local-test-server . >/dev/null
docker build -f Dockerfile.client -t sy-vpn-local-test-client . >/dev/null

echo "=== network + server ==="
docker network create sy-vpn-test-net >/dev/null 2>&1 || true
docker rm -f sy-vpn-server-test >/dev/null 2>&1 || true
docker run -d --name sy-vpn-server-test --network sy-vpn-test-net \
  --cap-add=NET_ADMIN --sysctl net.ipv4.ip_forward=1 --device=/dev/net/tun \
  sy-vpn-local-test-server >/dev/null
sleep 2
docker logs sy-vpn-server-test

echo ""
echo "=== running client ==="
if $SCOPED; then
  docker run --rm --network sy-vpn-test-net --cap-add=NET_ADMIN --device=/dev/net/tun \
    -e ALLOWED_IPS=10.66.0.0/16 sy-vpn-local-test-client
else
  docker run --rm --network sy-vpn-test-net --privileged --device=/dev/net/tun \
    sy-vpn-local-test-client
fi

if $KEEP; then
  echo ""
  echo "Server left running: docker logs -f sy-vpn-server-test / docker exec -it sy-vpn-server-test sh"
  echo "Tear down later with: docker rm -f sy-vpn-server-test && docker network rm sy-vpn-test-net"
else
  echo ""
  echo "=== cleanup ==="
  docker rm -f sy-vpn-server-test >/dev/null
  docker network rm sy-vpn-test-net >/dev/null
  rm -f sy-vpn-server
fi
