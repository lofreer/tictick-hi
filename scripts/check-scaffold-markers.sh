#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAILED=0

if rg -n 'PageStub' "$ROOT_DIR/web/frontend/src"; then
  echo
  echo "FAIL PageStub is still wired into the frontend. This is scaffold, not demo."
  FAILED=1
fi

if ! rg -q '^scaffold$' "$ROOT_DIR/README.md"; then
  echo
  echo "FAIL README must keep the current overall project status as scaffold."
  FAILED=1
fi

if ! rg -q 'tictick-hi 当前是 scaffold' "$ROOT_DIR/docs/quality-audit.md"; then
  echo
  echo "FAIL quality audit must keep the current overall project status as scaffold."
  FAILED=1
fi

if rg -n '^[[:space:]>*-]*(项目整体(达到|升级为|是|当前是) `?(usable|production-safe)`?|overall project (is|reached|upgraded to) (usable|production-safe))' "$ROOT_DIR/README.md" "$ROOT_DIR/docs/quality-audit.md"; then
  echo
  echo "FAIL overall usable / production-safe claims must not appear before audit closure."
  FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi
