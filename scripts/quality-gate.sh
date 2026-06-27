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
run_check "trading float64" "$ROOT_DIR/scripts/check-trading-floats.sh"
run_check "stage 0 scaffold markers" "$ROOT_DIR/scripts/check-scaffold-markers.sh"
run_audit "future risk markers" "$ROOT_DIR/scripts/check-future-risk-markers.sh"

if [ "$FAILED" -ne 0 ]; then
  echo "quality gate failed"
  exit 1
fi

echo "quality gate passed"
