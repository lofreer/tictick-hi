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
    printf 'stage1 real exchange data sync smoke requires %s\n' "$name" >&2
    exit 1
  fi
}

require_env POSTGRES_USER
require_env POSTGRES_PASSWORD
require_env POSTGRES_DB

STAMP="${STAGE1_REAL_EXCHANGE_SMOKE_STAMP:-$(date +%s)}"
SMOKE_DB="tictick_hi_real_exchange_smoke_${STAMP}"
BINANCE_BASE_URL="${TICTICK_REAL_BINANCE_BASE_URL:-https://data-api.binance.vision}"
SYMBOL="${TICTICK_REAL_EXCHANGE_SYMBOL:-BTCUSDT}"

cleanup() {
  docker compose exec -T postgres psql \
    -v ON_ERROR_STOP=1 \
    -v db="$SMOKE_DB" \
    -U "$POSTGRES_USER" \
    -d postgres <<'SQL' >/dev/null 2>&1 || true
SELECT pg_terminate_backend(pid)
  FROM pg_stat_activity
 WHERE datname = :'db'
   AND pid <> pg_backend_pid();
DROP DATABASE IF EXISTS :"db";
SQL
}
trap cleanup EXIT

docker compose up -d postgres >/dev/null
cleanup
docker compose exec -T postgres psql \
  -v ON_ERROR_STOP=1 \
  -v db="$SMOKE_DB" \
  -U "$POSTGRES_USER" \
  -d postgres <<'SQL' >/dev/null
CREATE DATABASE :"db";
SQL

docker run --rm \
  --network tictick-hi_default \
  -v "$ROOT_DIR":/src \
  -w /src \
  -e TICTICK_TEST_DATABASE_URL="postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${SMOKE_DB}?sslmode=disable" \
  -e TICTICK_REAL_EXCHANGE_SMOKE=1 \
  -e TICTICK_REAL_BINANCE_BASE_URL="$BINANCE_BASE_URL" \
  -e TICTICK_REAL_EXCHANGE_SYMBOL="$SYMBOL" \
  golang:1.26-bookworm \
  go test ./internal/web/api -run TestIntegrationRealBinanceDataSyncRouteServesNativeCandles -count=1 -v

printf '\nstage1 real exchange data sync smoke passed\n'
