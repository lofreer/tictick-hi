#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

TARGETS=(
  "$ROOT_DIR/internal/backtest"
  "$ROOT_DIR/internal/trading"
  "$ROOT_DIR/internal/data"
)

if rg -n 'float64|ParseFloat|FormatFloat' "${TARGETS[@]}"; then
  echo
  echo "FAIL trading facts must not use float64 / ParseFloat / FormatFloat in backtest, trading, or data packages."
  echo "Use an explicit decimal / money / quantity boundary before upgrading module status."
  exit 1
fi

