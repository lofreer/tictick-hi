#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FAILED=0

check_file() {
  local file="$1"
  local hard_limit="$2"
  local lines

  lines="$(wc -l < "$file" | tr -d ' ')"
  if [ "$lines" -gt "$hard_limit" ]; then
    printf 'FAIL file too large: %s has %s lines, hard limit %s\n' "$file" "$lines" "$hard_limit"
    FAILED=1
  fi
}

while IFS= read -r -d '' file; do
  case "$file" in
    */node_modules/*|*/dist/*|*/coverage/*|*/.git/*)
      continue
      ;;
    *_test.go)
      check_file "$file" 700
      ;;
    *.go)
      check_file "$file" 500
      ;;
    *.vue)
      check_file "$file" 450
      ;;
    *.test.ts)
      check_file "$file" 650
      ;;
    *.ts)
      check_file "$file" 400
      ;;
    *.css)
      check_file "$file" 500
      ;;
  esac
done < <(find "$ROOT_DIR" -type f \( -name '*.go' -o -name '*.ts' -o -name '*.vue' -o -name '*.css' \) -print0)

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi
