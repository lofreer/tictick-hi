#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
BIN="$TMP_DIR/hi"
FAILED=0
CASE_INDEX=0

SECRET_DSN='postgres://stage8_secret_user:stage8_secret_password@127.0.0.1:6543/tictick_hi?sslmode=disable'
FORBIDDEN_OUTPUT=(
  "$SECRET_DSN"
  "stage8_secret_password"
  "stage8_config_secret"
)

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cd "$ROOT_DIR"

go build -o "$BIN" ./cmd/hi

run_failing_case() {
  local name="$1"
  local expected="$2"
  shift 2
  local output="$TMP_DIR/case-$CASE_INDEX.out"
  CASE_INDEX=$((CASE_INDEX + 1))

  if "$@" >"$output" 2>&1; then
    echo "FAIL $name: command unexpectedly succeeded"
    cat "$output"
    FAILED=1
    return
  fi

  if ! grep -Fq "$expected" "$output"; then
    echo "FAIL $name: expected output to contain: $expected"
    cat "$output"
    FAILED=1
  fi

  for forbidden in "${FORBIDDEN_OUTPUT[@]}"; do
    if grep -Fq "$forbidden" "$output"; then
      echo "FAIL $name: output leaked forbidden secret marker: $forbidden"
      cat "$output"
      FAILED=1
    fi
  done
}

clean_env() {
  env -i PATH="$PATH" HOME="${HOME:-$TMP_DIR}" "$@"
}

run_failing_case \
  "sync missing database url" \
  "DATABASE_URL is required" \
  clean_env "$BIN" sync --once

run_failing_case \
  "invalid log level" \
  "LOG_LEVEL" \
  clean_env LOG_LEVEL=stage8_config_secret "$BIN" sync --once

run_failing_case \
  "invalid log format" \
  "LOG_FORMAT" \
  clean_env LOG_FORMAT=stage8_config_secret "$BIN" sync --once

run_failing_case \
  "invalid log correlation id" \
  "LOG_CORRELATION_ID" \
  clean_env LOG_CORRELATION_ID=stage8_config_secret! "$BIN" sync --once

run_failing_case \
  "invalid log traceparent" \
  "LOG_TRACEPARENT" \
  clean_env LOG_TRACEPARENT=stage8_config_secret "$BIN" sync --once

run_failing_case \
  "invalid database max conns" \
  "DB_MAX_CONNS" \
  clean_env DATABASE_URL="$SECRET_DSN" DB_MAX_CONNS=0 "$BIN" sync --once

run_failing_case \
  "sync invalid duration" \
  "SYNC_POLL_INTERVAL" \
  clean_env DATABASE_URL="$SECRET_DSN" SYNC_POLL_INTERVAL=not-a-duration "$BIN" sync --once

run_failing_case \
  "sync heartbeat longer than lease" \
  "SYNC_HEARTBEAT_INTERVAL" \
  clean_env DATABASE_URL="$SECRET_DSN" SYNC_LEASE_TTL=10s SYNC_HEARTBEAT_INTERVAL=11s "$BIN" sync --once

run_failing_case \
  "api invalid bool" \
  "AUTH_COOKIE_SECURE" \
  clean_env DATABASE_URL="$SECRET_DSN" AUTH_COOKIE_SECURE=stage8_config_secret "$BIN" api

run_failing_case \
  "sync invalid exchange limit" \
  "BINANCE_REQUEST_WEIGHT_LIMIT" \
  clean_env DATABASE_URL="$SECRET_DSN" BINANCE_REQUEST_WEIGHT_LIMIT=0 "$BIN" sync --once

run_failing_case \
  "sync invalid health addr" \
  "SYNC_HEALTH_ADDR" \
  clean_env DATABASE_URL="$SECRET_DSN" SYNC_HEALTH_ADDR=not-a-host-port "$BIN" sync --once

run_failing_case \
  "sync invalid ready backlog" \
  "SYNC_READY_MAX_BACKLOG" \
  clean_env DATABASE_URL="$SECRET_DSN" SYNC_READY_MAX_BACKLOG=-1 "$BIN" sync --once

run_failing_case \
  "sync invalid ready age" \
  "SYNC_READY_MAX_AGE" \
  clean_env DATABASE_URL="$SECRET_DSN" SYNC_READY_MAX_AGE=stage8_config_secret "$BIN" sync --once

run_failing_case \
  "sync invalid ready stale leases" \
  "SYNC_READY_MAX_STALE_LEASES" \
  clean_env DATABASE_URL="$SECRET_DSN" SYNC_READY_MAX_STALE_LEASES=-1 "$BIN" sync --once

run_failing_case \
  "sync invalid ready exchange backoffs" \
  "SYNC_READY_MAX_EXCHANGE_BACKOFFS" \
  clean_env DATABASE_URL="$SECRET_DSN" SYNC_READY_MAX_EXCHANGE_BACKOFFS=-1 "$BIN" sync --once

run_failing_case \
  "sync invalid ready catalog staleness" \
  "SYNC_READY_MAX_CATALOG_STALENESS" \
  clean_env DATABASE_URL="$SECRET_DSN" SYNC_READY_MAX_CATALOG_STALENESS=stage8_config_secret "$BIN" sync --once

run_failing_case \
  "trading invalid candle limit" \
  "TRADING_CANDLE_LIMIT" \
  clean_env DATABASE_URL="$SECRET_DSN" TRADING_CANDLE_LIMIT=0 "$BIN" trading --once

run_failing_case \
  "notify invalid retry delay" \
  "NOTIFY_RETRY_DELAY" \
  clean_env DATABASE_URL="$SECRET_DSN" NOTIFY_RETRY_DELAY=-1s "$BIN" notify --once

run_failing_case \
  "notify invalid provider config readiness" \
  "NOTIFY_READY_VALIDATE_PROVIDER_CONFIG" \
  clean_env DATABASE_URL="$SECRET_DSN" NOTIFY_READY_VALIDATE_PROVIDER_CONFIG=maybe "$BIN" notify --once

run_failing_case \
  "notify unknown flag" \
  "flag provided but not defined" \
  clean_env "$BIN" notify --unknown

if [ "$FAILED" -ne 0 ]; then
  echo "stage8 command config smoke failed"
  exit 1
fi

echo "stage8 command config smoke passed"
