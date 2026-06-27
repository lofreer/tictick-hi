#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

fail() {
  printf 'FAIL migration audit: %s\n' "$1" >&2
  exit 1
}

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    fail "$name is required"
  fi
}

psql_exec() {
  docker compose exec -T postgres psql \
    -v ON_ERROR_STOP=1 \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    "$@"
}

assert_migration_applied() {
  local version="$1"
  local applied
  applied="$(psql_exec -At -v version="$version" <<'SQL' | tr -d '[:space:]'
SELECT EXISTS (
  SELECT 1
    FROM schema_migrations
   WHERE version = :'version'
);
SQL
)"
  if [ "$applied" != "t" ]; then
    fail "missing schema_migrations entry $version"
  fi
}

assert_zero() {
  local label="$1"
  local sql="$2"
  local count
  count="$(psql_exec -At -c "$sql" | tr -d '[:space:]')"
  if [ "$count" != "0" ]; then
    fail "$label has $count violating rows"
  fi
}

assert_constraint_validated() {
  local constraint="$1"
  local validated
  validated="$(psql_exec -At -v constraint="$constraint" <<'SQL' | tr -d '[:space:]'
SELECT COALESCE(
  (
    SELECT convalidated
      FROM pg_constraint
     WHERE conname = :'constraint'
  ),
  false
);
SQL
)"
  if [ "$validated" != "t" ]; then
    fail "$constraint is not validated"
  fi
}

assert_trigger_enabled() {
  local table="$1"
  local trigger="$2"
  local enabled
  enabled="$(psql_exec -At -v table="$table" -v trigger="$trigger" <<'SQL' | tr -d '[:space:]'
SELECT EXISTS (
  SELECT 1
    FROM pg_trigger trigger
    JOIN pg_class relation ON relation.oid = trigger.tgrelid
   WHERE relation.relname = :'table'
     AND trigger.tgname = :'trigger'
     AND NOT trigger.tgisinternal
     AND trigger.tgenabled <> 'D'
);
SQL
)"
  if [ "$enabled" != "t" ]; then
    fail "$table trigger $trigger is not enabled"
  fi
}

require_env POSTGRES_USER
require_env POSTGRES_DB

while IFS= read -r migration; do
  assert_migration_applied "$(basename "$migration")"
done < <(find "$ROOT_DIR/internal/store/postgres/migrations" -maxdepth 1 -type f -name '*.sql' | sort)

assert_constraint_validated "data_sync_tasks_lease_consistency_check"
assert_constraint_validated "backtest_tasks_lease_consistency_check"
assert_constraint_validated "trading_tasks_lease_consistency_check"
assert_constraint_validated "notification_outbox_lease_consistency_check"

assert_trigger_enabled "data_sync_tasks" "data_sync_tasks_status_transition_guard"
assert_trigger_enabled "backtest_tasks" "backtest_tasks_status_transition_guard"
assert_trigger_enabled "trading_tasks" "trading_tasks_status_transition_guard"

assert_zero "data_sync terminal rows without finished_at" \
  "SELECT count(*) FROM data_sync_tasks WHERE status IN ('succeeded', 'failed', 'cancelled') AND finished_at IS NULL"

assert_zero "backtest terminal rows without finished_at" \
  "SELECT count(*) FROM backtest_tasks WHERE status IN ('succeeded', 'failed', 'cancelled') AND finished_at IS NULL"

assert_zero "trading terminal rows without finished_at" \
  "SELECT count(*) FROM trading_tasks WHERE status IN ('succeeded', 'failed', 'cancelled') AND finished_at IS NULL"

assert_zero "data_sync inconsistent lease rows" \
  "SELECT count(*) FROM data_sync_tasks WHERE NOT ((locked_by IS NULL AND locked_until IS NULL AND heartbeat_at IS NULL) OR (status = 'running' AND locked_by IS NOT NULL AND locked_until IS NOT NULL AND heartbeat_at IS NOT NULL))"

assert_zero "backtest inconsistent lease rows" \
  "SELECT count(*) FROM backtest_tasks WHERE NOT ((locked_by IS NULL AND locked_until IS NULL AND heartbeat_at IS NULL) OR (status = 'running' AND locked_by IS NOT NULL AND locked_until IS NOT NULL AND heartbeat_at IS NOT NULL))"

assert_zero "trading inconsistent lease rows" \
  "SELECT count(*) FROM trading_tasks WHERE NOT ((locked_by IS NULL AND locked_until IS NULL AND heartbeat_at IS NULL) OR (status = 'running' AND locked_by IS NOT NULL AND locked_until IS NOT NULL AND heartbeat_at IS NOT NULL))"

assert_zero "notification_outbox inconsistent lease rows" \
  "SELECT count(*) FROM notification_outbox WHERE NOT ((locked_by IS NULL AND locked_until IS NULL) OR (status = 'running' AND locked_by IS NOT NULL AND locked_until IS NOT NULL))"

assert_zero "strategy_intents missing or mismatched parent task rows" \
  "SELECT count(*) FROM strategy_intents si WHERE (si.task_type = 'backtest' AND NOT EXISTS (SELECT 1 FROM backtest_tasks bt WHERE bt.id = si.task_id)) OR (si.task_type IN ('paper', 'live') AND NOT EXISTS (SELECT 1 FROM trading_tasks tt WHERE tt.id = si.task_id AND tt.type = si.task_type))"

assert_zero "orders missing trading task rows" \
  "SELECT count(*) FROM orders o WHERE NOT EXISTS (SELECT 1 FROM trading_tasks tt WHERE tt.id = o.task_id AND tt.type = o.task_type)"

assert_zero "orders missing intent rows" \
  "SELECT count(*) FROM orders o WHERE o.intent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM strategy_intents si WHERE si.id = o.intent_id AND si.task_id = o.task_id)"

assert_zero "executions missing parent rows" \
  "SELECT count(*) FROM executions e WHERE NOT EXISTS (SELECT 1 FROM trading_tasks tt WHERE tt.id = e.task_id AND tt.type = e.task_type) OR NOT EXISTS (SELECT 1 FROM orders o WHERE o.id = e.order_id AND o.task_id = e.task_id) OR (e.intent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM strategy_intents si WHERE si.id = e.intent_id AND si.task_id = e.task_id))"

assert_zero "positions missing trading task rows" \
  "SELECT count(*) FROM positions p WHERE NOT EXISTS (SELECT 1 FROM trading_tasks tt WHERE tt.id = p.task_id AND tt.type = p.task_type)"

assert_zero "notifications missing parent rows" \
  "SELECT count(*) FROM notifications n WHERE NOT EXISTS (SELECT 1 FROM trading_tasks tt WHERE tt.id = n.task_id) OR (n.intent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM strategy_intents si WHERE si.id = n.intent_id AND si.task_id = n.task_id))"

assert_zero "notification_outbox missing parent rows" \
  "SELECT count(*) FROM notification_outbox no WHERE NOT EXISTS (SELECT 1 FROM notifications n WHERE n.id = no.notification_id AND n.task_id = no.task_id) OR NOT EXISTS (SELECT 1 FROM trading_tasks tt WHERE tt.id = no.task_id) OR (no.intent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM strategy_intents si WHERE si.id = no.intent_id AND si.task_id = no.task_id))"

assert_zero "backtest_orders missing parent rows" \
  "SELECT count(*) FROM backtest_orders bo WHERE NOT EXISTS (SELECT 1 FROM backtest_tasks bt WHERE bt.id = bo.backtest_id) OR (bo.intent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM strategy_intents si WHERE si.id = bo.intent_id AND si.task_id = bo.backtest_id))"

echo "stage8 migration audit passed"
