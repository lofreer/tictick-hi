#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

FAILED=0

run_check() {
  local title="$1"
  local command="$2"

  echo "== quality gate: $title =="
  if ! "$command"; then
    FAILED=1
  fi
  echo
}

run_audit() {
  local title="$1"
  local command="$2"

  echo "== quality audit: $title =="
  if ! "$command"; then
    echo "audit findings retained for later stages"
  fi
  echo
}

run_check "file size" "$ROOT_DIR/scripts/check-file-size.sh"
run_check "stage 0 foundation" "$ROOT_DIR/scripts/check-stage0-foundation.sh"
run_check "trading float64" "$ROOT_DIR/scripts/check-trading-floats.sh"
run_check "strategy boundary" "$ROOT_DIR/scripts/check-strategy-boundary.sh"
run_check "api contract drift" "$ROOT_DIR/scripts/check-api-contract-drift.sh"
run_check "research chart layout" "$ROOT_DIR/scripts/check-research-chart-layout.sh"
run_check "command config smoke" "$ROOT_DIR/scripts/stage8-command-config-smoke.sh"
run_check "capacity check" "$ROOT_DIR/scripts/stage8-capacity-check.sh"
run_check "backup dry run" "$ROOT_DIR/scripts/stage8-backup-dry-run-smoke.sh"
run_check "stage 0 scaffold markers" "$ROOT_DIR/scripts/check-scaffold-markers.sh"
run_audit "future risk markers" "$ROOT_DIR/scripts/check-future-risk-markers.sh"

if [ "$FAILED" -ne 0 ]; then
  echo "quality gate failed"
  exit 1
fi

echo "quality gate passed"
