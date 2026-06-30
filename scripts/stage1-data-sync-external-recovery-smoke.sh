#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    printf 'stage1 data sync external recovery smoke requires %s\n' "$name" >&2
    exit 1
  fi
}

require_env POSTGRES_USER
require_env POSTGRES_PASSWORD
require_env POSTGRES_DB

docker compose up -d postgres >/dev/null

docker run --rm \
  --network tictick-hi_default \
  -v "$ROOT_DIR":/src \
  -w /src \
  -e TICTICK_TEST_DATABASE_URL="postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable" \
  golang:1.26-bookworm \
  go test ./internal/store/postgres -run 'TestIntegrationDataSyncRunnerRecoversAfter(BinanceRetryAfter|OKXRateLimitCode)' -count=1 -v

printf '\nstage1 data sync external recovery smoke passed\n'
