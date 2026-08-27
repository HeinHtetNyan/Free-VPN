#!/usr/bin/env bash
# Cross-compiles everything infra/scripts/setup-amnezia-interface.sh needs
# onto the central VPS: the AmneziaWG userspace daemon itself (built from
# pinned upstream source, not a downloaded binary — no official release
# binaries are published) plus this repo's own amnezia-setup/gen-params
# helpers (infra/amnezia-setup/). Uses Docker so this doesn't require a Go
# toolchain on whatever machine runs it, matching infra/local-test/run.sh.
#
# Output goes to infra/build/ (gitignored — see infra/local-test/.gitignore
# for the same pattern): amneziawg-go, amnezia-setup, gen-params.
set -euo pipefail
cd "$(dirname "$0")/.."

# Pinned to a specific commit rather than a branch tip for reproducibility —
# bump deliberately, not implicitly on every rebuild. Checked 2026-08-27.
AMNEZIAWG_GO_COMMIT="1b86b2ae0e493e7ea93f8c1a0f0cb6735b1551f1"

mkdir -p build
docker volume create sy-vpn-amnezia-gocache >/dev/null

echo "=== building amneziawg-go (upstream, pinned @ ${AMNEZIAWG_GO_COMMIT:0:12}) ==="
docker run --rm -v "$(pwd)/build":/out -v sy-vpn-amnezia-gocache:/go/pkg/mod golang:1.25 sh -c "
  set -e
  git clone -q https://github.com/amnezia-vpn/amneziawg-go /src
  cd /src
  git checkout -q ${AMNEZIAWG_GO_COMMIT}
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/amneziawg-go .
"

echo "=== building infra/amnezia-setup (amnezia-setup, gen-params) ==="
docker run --rm -v "$(pwd)/amnezia-setup":/src -v "$(pwd)/build":/out -v sy-vpn-amnezia-gocache:/go/pkg/mod -w /src golang:1.25 sh -c "
  set -e
  go mod tidy
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/amnezia-setup ./cmd/amnezia-setup
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/gen-params ./cmd/gen-params
"

echo "=== done: $(ls build) ==="
