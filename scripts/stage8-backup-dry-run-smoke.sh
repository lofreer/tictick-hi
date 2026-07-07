#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

POSTGRES_USER=stage8 \
POSTGRES_DB=tictick_hi \
STAGE8_BACKUP_STAMP=20260101T000000Z \
STAGE8_BACKUP_RETENTION_DAYS=7 \
  scripts/stage8-backup.sh --dry-run >/dev/null

if POSTGRES_USER=stage8 \
  POSTGRES_DB=tictick_hi \
  STAGE8_BACKUP_RETENTION_DAYS=0 \
  scripts/stage8-backup.sh --dry-run >/dev/null 2>&1; then
  echo "FAIL backup dry-run smoke: invalid retention unexpectedly passed"
  exit 1
fi

echo "stage8 backup dry-run smoke passed"
