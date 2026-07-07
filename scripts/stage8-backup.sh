#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

DRY_RUN=0
if [ "${1:-}" = "--dry-run" ]; then
  DRY_RUN=1
  shift
fi
if [ "$#" -ne 0 ]; then
  echo "usage: scripts/stage8-backup.sh [--dry-run]" >&2
  exit 2
fi

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

fail() {
  printf 'FAIL backup: %s\n' "$1" >&2
  exit 1
}

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    fail "$name is required"
  fi
}

positive_int() {
  local name="$1"
  local value="$2"
  if ! [[ "$value" =~ ^[0-9]+$ ]] || [ "$value" -le 0 ]; then
    fail "$name must be a positive integer"
  fi
}

validate_stamp() {
  local stamp="$1"
  if [[ ! "$stamp" =~ ^[A-Za-z0-9._:-]+$ ]]; then
    fail "STAGE8_BACKUP_STAMP contains unsupported characters"
  fi
}

wait_for_postgres() {
  local attempt
  for attempt in $(seq 1 60); do
    if docker compose exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
      return
    fi
    sleep 1
  done
  fail "postgres did not become ready"
}

require_env POSTGRES_USER
require_env POSTGRES_DB

BACKUP_DIR="${STAGE8_BACKUP_DIR:-../tictick-hi-backups}"
RETENTION_DAYS="${STAGE8_BACKUP_RETENTION_DAYS:-7}"
STAMP="${STAGE8_BACKUP_STAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
BACKUP_FILE="$BACKUP_DIR/tictick-hi-$STAMP.dump"

positive_int STAGE8_BACKUP_RETENTION_DAYS "$RETENTION_DAYS"
validate_stamp "$STAMP"

if [ "$DRY_RUN" -eq 1 ]; then
  printf 'stage8 backup dry run passed: backup_file=%s retention_days=%s\n' "$BACKUP_FILE" "$RETENTION_DAYS"
  exit 0
fi

mkdir -p "$BACKUP_DIR"

TMP_FILE="$BACKUP_FILE.tmp"
cleanup() {
  rm -f "$TMP_FILE"
}
trap cleanup EXIT

docker compose up -d postgres >/dev/null
wait_for_postgres

docker compose exec -T postgres pg_dump \
  -U "$POSTGRES_USER" \
  -d "$POSTGRES_DB" \
  -Fc \
  > "$TMP_FILE"

if [ ! -s "$TMP_FILE" ]; then
  fail "pg_dump produced an empty dump"
fi

mv "$TMP_FILE" "$BACKUP_FILE"
find "$BACKUP_DIR" -type f -name 'tictick-hi-*.dump' -mtime +"$RETENTION_DAYS" -print -delete

printf 'stage8 backup passed: backup_file=%s retention_days=%s\n' "$BACKUP_FILE" "$RETENTION_DAYS"
