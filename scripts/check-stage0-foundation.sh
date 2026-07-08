#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAILED=0

fail() {
  echo "FAIL $1"
  FAILED=1
}

check_file_max_lines() {
  local file="$1"
  local max_lines="$2"
  local lines

  lines="$(wc -l < "$ROOT_DIR/$file" | tr -d ' ')"
  if [ "$lines" -gt "$max_lines" ]; then
    fail "$file has $lines lines, stage 0 split limit is $max_lines"
  fi
}

check_file_exists() {
  local file="$1"

  if [ ! -f "$ROOT_DIR/$file" ]; then
    fail "$file is missing"
  fi
}

check_rg() {
  local pattern="$1"
  local file="$2"
  local message="$3"

  if ! rg -q "$pattern" "$ROOT_DIR/$file"; then
    fail "$message"
  fi
}

check_no_rg() {
  local pattern="$1"
  local message="$2"
  shift 2

  local paths=()
  local path
  for path in "$@"; do
    paths+=("$ROOT_DIR/$path")
  done

  if rg -n "$pattern" "${paths[@]}"; then
    fail "$message"
  fi
}

check_file_exists "internal/web/api/server.go"
check_file_exists "web/frontend/src/i18n/messages.ts"
check_file_exists "web/frontend/src/i18n/messages.zh.ts"
check_file_exists "web/frontend/src/i18n/messages.en.ts"
check_file_exists "web/frontend/src/i18n/messages.research.zh.ts"
check_file_exists "web/frontend/src/i18n/messages.research.en.ts"

check_file_max_lines "internal/web/api/server.go" 220
check_file_max_lines "web/frontend/src/i18n/messages.ts" 80

check_no_rg 'PageStub' "PageStub must not be wired into routes or pages" "web/frontend/src/router" "web/frontend/src/pages"

check_rg '^scaffold$' "README.md" "README must keep the current overall project status as scaffold"
check_rg 'docs/ai-delivery-protocol.md' "README.md" "README must link the AI delivery protocol"
check_rg 'docs/quality-audit.md' "README.md" "README must link the quality audit"
check_rg 'docs/implementation-plan.md' "README.md" "README must link the implementation plan"
check_rg 'tictick-local-ops-2026' "README.md" "README must document the password-policy-compliant local operator password"
check_rg 'BOOTSTRAP_OPERATOR_PASSWORD=tictick-local-ops-2026' ".env.example" ".env.example must keep the password-policy-compliant local operator password"
check_no_rg 'password: tictick-local-admin-password|BOOTSTRAP_OPERATOR_PASSWORD=tictick-local-admin-password' "old local operator password must not return to local runbook config because it violates the password policy" "README.md" ".env.example"
check_no_rg '"tictick-local-admin-password"' "old local operator password must not return to browser smoke defaults because it violates the password policy" "scripts/research-chart-height-smoke.mjs" "scripts/stage8-state-visual-smoke.mjs" "scripts/stage8-visual-smoke.mjs"
check_rg 'tictick-hi 当前是 scaffold' "docs/quality-audit.md" "quality audit must keep the current overall project status as scaffold"
check_rg '### 阶段 0 Definition of Done：质量底座' "docs/quality-audit.md" "quality audit must keep the stage 0 Definition of Done"

check_rg 'check-file-size.sh' "scripts/quality-gate.sh" "quality gate must keep the file size check"
check_rg 'check-scaffold-markers.sh' "scripts/quality-gate.sh" "quality gate must keep scaffold marker checks"
check_rg 'check-stage0-foundation.sh' "scripts/quality-gate.sh" "quality gate must include the stage 0 foundation check"

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

echo "stage 0 foundation check passed"
