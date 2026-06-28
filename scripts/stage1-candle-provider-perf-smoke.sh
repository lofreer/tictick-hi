#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [ -f .env.example ]; then
  set -a
  source .env.example
  set +a
fi
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

log() {
  printf '\n== stage1 candle provider perf smoke: %s ==\n' "$1"
}

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    fail "$name is required"
  fi
}

default_env_from_example() {
  local name="$1"
  if [ -n "${!name:-}" ] || [ ! -f .env.example ]; then
    return 0
  fi
  local value
  value="$(awk -F= -v key="$name" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' .env.example)"
  if [ -n "$value" ]; then
    printf -v "$name" "%s" "$value"
    export "$name"
  fi
}

default_env_from_example POSTGRES_USER
default_env_from_example POSTGRES_PASSWORD
default_env_from_example POSTGRES_DB
default_env_from_example ENCRYPTION_KEY

require_env POSTGRES_USER
require_env POSTGRES_PASSWORD
require_env POSTGRES_DB
require_env ENCRYPTION_KEY

DATABASE_URL="postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"
PERF_MAX_MS="${TICTICK_CANDLE_PERF_MAX_MS:-10000}"
GO_IMAGE="${TICTICK_GO_TEST_IMAGE:-golang:1.26-bookworm}"

wait_for_postgres() {
  local deadline
  deadline=$((SECONDS + 60))
  while [ "$SECONDS" -lt "$deadline" ]; do
    if docker compose exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  fail "postgres did not become ready"
}

log "start postgres"
docker compose up -d postgres
wait_for_postgres

log "run large aggregation query test"
docker run --rm \
  --network tictick-hi_default \
  -v "$ROOT_DIR":/src \
  -w /src \
  -e TICTICK_TEST_DATABASE_URL="$DATABASE_URL" \
  -e TICTICK_CANDLE_PERF_MAX_MS="$PERF_MAX_MS" \
  -e ENCRYPTION_KEY="$ENCRYPTION_KEY" \
  "$GO_IMAGE" \
  go test ./internal/store/postgres \
    -run TestIntegrationCandleProviderLargeAggregationWindowPerformance \
    -count=1 \
    -v

log "passed"
