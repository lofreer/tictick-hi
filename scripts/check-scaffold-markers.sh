#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAILED=0

if rg -n 'PageStub' "$ROOT_DIR/web/frontend/src"; then
  echo
  echo "FAIL PageStub is still wired into the frontend. This is scaffold, not demo."
  FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi
