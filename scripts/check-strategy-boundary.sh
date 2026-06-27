#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

imports="$(go list -f '{{range .Imports}}{{.}}{{"\n"}}{{end}}' ./internal/strategy)"

if printf '%s\n' "$imports" | rg -n '(^net/http$|^database/sql$|^os/exec$|github\.com/jackc/pgx|github\.com/lofreer/tictick-hi/internal/(store|web|trading|backtest|datasync))'; then
  echo
  echo "FAIL strategies must stay pure: no store/web/trading/backtest/datasync/database/http side effects."
  exit 1
fi
