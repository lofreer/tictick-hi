#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAILED=0

require_dir() {
  local path="$1"
  if [ ! -d "$ROOT_DIR/$path" ]; then
    echo "FAIL required directory missing: $path"
    FAILED=1
  fi
}

reject_dir() {
  local path="$1"
  if [ -d "$ROOT_DIR/$path" ]; then
    echo "FAIL unexpected duplicate project root directory exists: $path"
    FAILED=1
  fi
}

required_dirs=(
  "cmd/hi"
  "internal/adapter"
  "internal/backtest"
  "internal/data"
  "internal/datasync"
  "internal/exchange"
  "internal/marketsync"
  "internal/notification"
  "internal/secretbox"
  "internal/store/postgres"
  "internal/strategy"
  "internal/trading"
  "internal/web/api"
  "internal/workerlease"
  "internal/workerlog"
  "web/frontend/src/app"
  "web/frontend/src/assets"
  "web/frontend/src/components/chart"
  "web/frontend/src/components/common"
  "web/frontend/src/components/layout"
  "web/frontend/src/components/tables"
  "web/frontend/src/composables"
  "web/frontend/src/i18n"
  "web/frontend/src/pages"
  "web/frontend/src/router"
  "web/frontend/src/services/api"
  "web/frontend/src/stores"
  "web/frontend/src/styles"
  "web/frontend/src/theme"
  "web/frontend/src/types"
  "web/frontend/src/utils"
  "docs"
  "scripts"
  "deploy/systemd"
)

for path in "${required_dirs[@]}"; do
  require_dir "$path"
done

duplicate_root_dirs=(
  "api"
  "backend"
  "client"
  "frontend"
  "server"
)

for path in "${duplicate_root_dirs[@]}"; do
  reject_dir "$path"
done

if find "$ROOT_DIR/internal" "$ROOT_DIR/cmd" -type f \( -name '*.vue' -o -name '*.ts' -o -name '*.tsx' \) | grep -q .; then
  echo "FAIL frontend source files must stay under web/frontend/src"
  FAILED=1
fi

if find "$ROOT_DIR/web/frontend/src" -type f -name '*.go' | grep -q .; then
  echo "FAIL Go source files must stay outside web/frontend/src"
  FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

echo "repo structure check passed"
