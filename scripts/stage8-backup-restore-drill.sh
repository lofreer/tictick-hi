#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

STAMP="${STAGE8_BACKUP_RESTORE_STAMP:-$(date +%s)}"
DRILL_DB="${STAGE8_BACKUP_RESTORE_DB:-tictick_hi_restore_drill_${STAMP}}"
TMP_DIR="$(mktemp -d)"
DUMP_FILE="$TMP_DIR/source.dump"
SOURCE_TABLES_FILE="$TMP_DIR/source-tables.txt"
RESTORED_TABLES_FILE="$TMP_DIR/restored-tables.txt"
DRILL_DB_CREATED=0

log() {
  printf '\n== stage8 backup restore drill: %s ==\n' "$1"
}

fail() {
  printf 'FAIL backup restore drill: %s\n' "$1" >&2
  exit 1
}

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    fail "$name is required"
  fi
}

validate_db_identifier() {
  local name="$1"
  local value="$2"
  if [[ ! "$value" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
    fail "$name must be a simple PostgreSQL identifier, got $value"
  fi
}

url_encode() {
  node -e 'process.stdout.write(encodeURIComponent(process.argv[1]))' "$1"
}

drill_database_url() {
  local user password database
  user="$(url_encode "$POSTGRES_USER")"
  password="$(url_encode "$POSTGRES_PASSWORD")"
  database="$(url_encode "$DRILL_DB")"
  printf 'postgresql://%s:%s@postgres:5432/%s?sslmode=disable' "$user" "$password" "$database"
}

psql_query() {
  local database="$1"
  shift
  docker compose exec -T postgres psql \
    -v ON_ERROR_STOP=1 \
    -U "$POSTGRES_USER" \
    -d "$database" \
    -At \
    "$@"
}

drop_drill_db() {
  docker compose exec -T postgres psql \
    -v ON_ERROR_STOP=1 \
    -U "$POSTGRES_USER" \
    -d postgres \
    -v drill_db="$DRILL_DB" <<'SQL' >/dev/null
SELECT pg_terminate_backend(pid)
  FROM pg_stat_activity
 WHERE datname = :'drill_db'
   AND pid <> pg_backend_pid();

DROP DATABASE IF EXISTS :"drill_db";
SQL
}

cleanup() {
  local status=$?
  if [ "$DRILL_DB_CREATED" -eq 1 ] && [ "${STAGE8_BACKUP_RESTORE_KEEP_DB:-0}" != "1" ]; then
    drop_drill_db || true
  fi
  rm -rf "$TMP_DIR"
  exit "$status"
}
trap cleanup EXIT

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

write_table_list() {
  local database="$1"
  local output_file="$2"
  psql_query "$database" <<'SQL' > "$output_file"
SELECT tablename
  FROM pg_tables
 WHERE schemaname = 'public'
 ORDER BY tablename;
SQL
}

assert_same_table_list() {
  if ! diff -u "$SOURCE_TABLES_FILE" "$RESTORED_TABLES_FILE" >/dev/null; then
    printf 'source/restored table list mismatch:\n' >&2
    diff -u "$SOURCE_TABLES_FILE" "$RESTORED_TABLES_FILE" >&2 || true
    fail "restored table list does not match source"
  fi
}

assert_all_migrations_applied() {
  local migration version applied
  while IFS= read -r migration; do
    version="$(basename "$migration")"
    applied="$(psql_query "$DRILL_DB" -v version="$version" <<'SQL' | tr -d '[:space:]'
SELECT EXISTS (
  SELECT 1
    FROM schema_migrations
   WHERE version = :'version'
);
SQL
)"
    if [ "$applied" != "t" ]; then
      fail "restored database is missing schema_migrations entry $version"
    fi
  done < <(find "$ROOT_DIR/internal/store/postgres/migrations" -maxdepth 1 -type f -name '*.sql' | sort)
}

require_env POSTGRES_USER
require_env POSTGRES_PASSWORD
require_env POSTGRES_DB
validate_db_identifier "STAGE8_BACKUP_RESTORE_DB" "$DRILL_DB"

log "start postgres"
docker compose up -d postgres >/dev/null
wait_for_postgres

log "migrate source database"
docker compose run --rm migrate >/dev/null

log "capture source schema table list"
write_table_list "$POSTGRES_DB" "$SOURCE_TABLES_FILE"
if [ ! -s "$SOURCE_TABLES_FILE" ]; then
  fail "source database has no public tables after migration"
fi

log "dump source database"
docker compose exec -T postgres pg_dump \
  -U "$POSTGRES_USER" \
  -d "$POSTGRES_DB" \
  -Fc \
  > "$DUMP_FILE"
if [ ! -s "$DUMP_FILE" ]; then
  fail "pg_dump produced an empty dump"
fi

log "prepare drill database $DRILL_DB"
drop_drill_db
docker compose exec -T postgres createdb -U "$POSTGRES_USER" "$DRILL_DB"
DRILL_DB_CREATED=1

log "restore dump into drill database"
docker compose exec -T postgres pg_restore \
  --exit-on-error \
  --clean \
  --if-exists \
  -U "$POSTGRES_USER" \
  -d "$DRILL_DB" \
  < "$DUMP_FILE"

log "verify migrate idempotence on drill database"
DRILL_DATABASE_URL="$(drill_database_url)"
docker compose run --rm -e DATABASE_URL="$DRILL_DATABASE_URL" migrate >/dev/null

log "verify restored schema"
write_table_list "$DRILL_DB" "$RESTORED_TABLES_FILE"
assert_same_table_list
assert_all_migrations_applied

if [ "${STAGE8_BACKUP_RESTORE_KEEP_DB:-0}" = "1" ]; then
  printf 'stage8 backup restore drill passed; kept drill database %s\n' "$DRILL_DB"
else
  printf 'stage8 backup restore drill passed\n'
fi
