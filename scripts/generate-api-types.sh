#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

TICTICK_WRITE_GENERATED_API_TYPES=1 go test ./internal/web/api -run '^TestWriteGeneratedFrontendAPITypes$' -count=1
