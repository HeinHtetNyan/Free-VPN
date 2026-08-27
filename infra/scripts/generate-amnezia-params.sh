#!/usr/bin/env bash
# Generates one matched set of AmneziaWG obfuscation parameters (AWG_*) and
# prints them ready to paste into backend/.env. Run this ONCE per deployment
# — see backend/internal/servers/amnezia.go for why regenerating on every
# restart would strand already-issued client configs (every client's config
# must carry the exact same values the server interface was configured
# with). Re-run only when deliberately rotating obfuscation params, which
# requires every currently-connected client to reconnect and fetch a fresh
# config.
#
# Runs infra/amnezia-setup's gen-params inside Docker (matching the
# container's own platform, not cross-compiled) so this works regardless of
# what machine invokes it — no local Go toolchain needed.
set -euo pipefail
cd "$(dirname "$0")/.."

docker volume create sy-vpn-amnezia-gocache >/dev/null
docker run --rm -v "$(pwd)/amnezia-setup":/src -v sy-vpn-amnezia-gocache:/go/pkg/mod -w /src golang:1.25 sh -c '
  go mod tidy >/dev/null
  go run ./cmd/gen-params
'
