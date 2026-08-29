#!/usr/bin/env bash
# Generates one WireGuard client config for a friend/beta tester and a QR
# code for it, by calling the real production API (no local backend/DB
# access needed) — the same /auth/register + /connect flow the Android app
# uses, just invoked directly instead of from the app.
#
# Each friend gets their own device/peer (own device_id -> own token -> own
# WireGuard keypair), so access can be told apart per person later even
# though there's no revoke endpoint yet (see docs/OPEN_QUESTIONS.md) — if
# that's ever needed, it's a peer_store.go / /connect addition, not a
# rebuild of this script.
#
# iOS import: production issues AmneziaWG-obfuscated configs (Jc/Jmin/Jmax/
# I1-I5 junk-packet fields — see docs/ARCHITECTURE.md "Censorship
# resistance"), which the plain official WireGuard app and generic clients
# like V2Box do NOT understand. Use the official "AmneziaWG" App Store app
# instead (id6478942365, published by the Amnezia team) — scan the QR with
# it, or paste the .conf contents in manually. There is no "v2ray link" for
# this; the config file itself is the interchange format.
#
# Usage: ./generate-friend-config.sh <friend-name> [location_id]
set -euo pipefail

FRIEND="${1:?usage: generate-friend-config.sh <friend-name> [location_id]}"
LOCATION_ID="${2:-singapore}"
API_BASE="${SY_API_BASE:-https://sy-api.malmah.fyi}"

OUT_DIR="$(dirname "$0")/../out/friends"
mkdir -p "$OUT_DIR"

DEVICE_ID="friend-${FRIEND}-$(head -c8 /dev/urandom | od -An -tx1 | tr -d ' \n')"

echo "==> Registering device for ${FRIEND}"
TOKEN=$(curl -sf -X POST "${API_BASE}/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"device_id\":\"${DEVICE_ID}\"}" | tr -d '\r' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

if [ -z "${TOKEN}" ]; then
  echo "error: registration did not return a token" >&2
  exit 1
fi

echo "==> Requesting config for location '${LOCATION_ID}'"
CONFIG=$(curl -sf -X POST "${API_BASE}/connect" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"location_id\":\"${LOCATION_ID}\"}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["config"])')

if [ -z "${CONFIG}" ]; then
  echo "error: /connect did not return a config" >&2
  exit 1
fi

CONF_PATH="${OUT_DIR}/${FRIEND}.conf"
PNG_PATH="${OUT_DIR}/${FRIEND}.png"

printf '%s\n' "${CONFIG}" > "${CONF_PATH}"
echo "==> Wrote ${CONF_PATH}"

if command -v qrencode >/dev/null 2>&1; then
  qrencode -t PNG -o "${PNG_PATH}" -r "${CONF_PATH}"
  echo "==> Wrote ${PNG_PATH} (scan with the official AmneziaWG iOS app)"
else
  echo "==> qrencode not installed, skipping QR image."
  echo "    Install with: sudo apt-get install qrencode"
  echo "    Or generate it later from ${CONF_PATH} with:"
  echo "      qrencode -t PNG -o ${PNG_PATH} -r ${CONF_PATH}"
fi

echo
echo "Give ${FRIEND} either:"
echo "  - the PNG (scan in the official AmneziaWG iOS app), or"
echo "  - the .conf contents (paste into the AmneziaWG app manually)"
echo "Treat both like a password — anyone who has them can use this VPN as ${FRIEND}."
