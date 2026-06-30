#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FAILED=0

run_check() {
  local title="$1"
  shift

  printf '\n== full quality gate: %s ==\n' "$title"
  if ! "$@"; then
    FAILED=1
  fi
}

env_enabled() {
  case "${1:-0}" in
    1|true|TRUE|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

run_check "go test" go test ./...
run_check "go vet" go vet ./...
run_check "frontend typecheck" pnpm --dir web/frontend run typecheck
run_check "frontend test" pnpm --dir web/frontend run test
run_check "frontend build" pnpm --dir web/frontend run build
run_check "lightweight quality gate" "$ROOT_DIR/scripts/quality-gate.sh"

if env_enabled "${FULL_QUALITY_STAGE1:-0}" || env_enabled "${FULL_QUALITY_STAGE1_RESTART:-0}"; then
  run_check "stage1 data sync restart smoke" "$ROOT_DIR/scripts/stage1-data-sync-restart-smoke.sh"
fi

if env_enabled "${FULL_QUALITY_STAGE1:-0}" || env_enabled "${FULL_QUALITY_STAGE1_EXTERNAL_RECOVERY:-0}"; then
  run_check "stage1 data sync external recovery smoke" "$ROOT_DIR/scripts/stage1-data-sync-external-recovery-smoke.sh"
fi

if env_enabled "${FULL_QUALITY_STAGE1:-0}" || env_enabled "${FULL_QUALITY_STAGE1_REAL_EXCHANGE:-0}"; then
  run_check "stage1 real exchange data sync smoke" "$ROOT_DIR/scripts/stage1-real-exchange-data-sync-smoke.sh"
fi

if env_enabled "${FULL_QUALITY_STAGE1:-0}" || env_enabled "${FULL_QUALITY_STAGE1_CANDLE_PERF:-0}"; then
  run_check "stage1 candle provider perf smoke" "$ROOT_DIR/scripts/stage1-candle-provider-perf-smoke.sh"
fi

if env_enabled "${FULL_QUALITY_STAGE8:-0}"; then
  run_check "stage8 full-chain smoke" "$ROOT_DIR/scripts/stage8-smoke.sh"
fi

if env_enabled "${FULL_QUALITY_SIGTERM:-0}"; then
  run_check "stage8 sigterm smoke" "$ROOT_DIR/scripts/stage8-sigterm-smoke.sh"
fi

if [ "$FAILED" -ne 0 ]; then
  printf '\nfull quality gate failed\n' >&2
  exit 1
fi

printf '\nfull quality gate passed\n'
