#!/usr/bin/env bash
set -euo pipefail

cd /home/appbox/SY

echo "==> Syncing to latest main"
# git reset --hard (not a plain pull) so this always converges even if the
# checkout drifted — this runs non-interactively over SSH from CI. Never
# touches untracked files, so backend/.env and backend/data/ are safe.
git fetch origin main
git reset --hard origin/main

echo "==> Rebuilding and restarting the API"
cd backend
docker compose up -d --build --remove-orphans

echo "==> Waiting for /health"
PORT=$(grep -E '^PORT=' .env 2>/dev/null | cut -d= -f2)
PORT="${PORT:-8080}"
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    echo "==> Healthy"
    break
  fi
  sleep 2
done

echo "==> Deploy complete"
docker compose ps
