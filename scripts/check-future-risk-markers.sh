#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FOUND=0

if rg -n 'secretDigest' "$ROOT_DIR/internal" "$ROOT_DIR/web/frontend/src"; then
  echo
  echo "AUDIT exchange account secrets are still handled by the legacy digest helper."
  FOUND=1
fi

if rg -n 'pending_submission' "$ROOT_DIR/internal/trading" "$ROOT_DIR/internal/store"; then
  echo
  echo "AUDIT live trading still stops at local pending_submission and is not a real live executor."
  FOUND=1
fi

if [ "$FOUND" -ne 0 ]; then
  exit 1
fi
