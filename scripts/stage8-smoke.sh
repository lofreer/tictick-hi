#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

BASE_URL="${BASE_URL:-http://127.0.0.1:${HTTP_PORT:-8080}}"
STAMP="${STAGE8_SMOKE_STAMP:-$(date +%s)}"
SYMBOL="S8${STAMP}USDT"
LOCK_SYMBOL="S8LOCK${STAMP}USDT"
CHANNEL="stage8-smoke-${STAMP}"
START_TIME="2026-01-01T00:00:00Z"
END_TIME="2026-01-01T02:00:00Z"

TMP_DIR="$(mktemp -d)"
COOKIE_JAR="$TMP_DIR/cookies.txt"
BODY_FILE="$TMP_DIR/body.json"
trap 'rm -rf "$TMP_DIR"' EXIT

log() {
  printf '\n== stage8 smoke: %s ==\n' "$1"
}

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  if [ -s "$BODY_FILE" ]; then
    printf 'last response:\n' >&2
    cat "$BODY_FILE" >&2
    printf '\n' >&2
  fi
  exit 1
}

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    fail "$name is required"
  fi
}

json_get() {
  local file="$1"
  local path="$2"
  node - "$file" "$path" <<'NODE'
const fs = require("fs");
const data = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
const parts = process.argv[3].split(".");
let value = data;
for (const part of parts) {
  if (part === "length") {
    value = Array.isArray(value) ? value.length : undefined;
  } else {
    value = value?.[part];
  }
}
if (value === undefined || value === null) process.exit(1);
if (typeof value === "object") console.log(JSON.stringify(value));
else console.log(String(value));
NODE
}

csrf_token() {
  awk '$6 == "tictick_hi_csrf" { token = $7 } END { print token }' "$COOKIE_JAR"
}

api_request() {
  local method="$1"
  local path="$2"
  local expected="$3"
  local payload="${4:-}"
  local token
  token="$(csrf_token)"
  local headers=(-H "Content-Type: application/json")
  if [ "$method" != "GET" ] && [ "$method" != "HEAD" ]; then
    headers+=(-H "X-CSRF-Token: $token")
  fi
  local args=(-sS -o "$BODY_FILE" -w "%{http_code}" -b "$COOKIE_JAR" -c "$COOKIE_JAR" -X "$method")
  if [ -n "$payload" ]; then
    args+=(--data "$payload")
  fi
  local code
  code="$(curl "${args[@]}" "${headers[@]}" "$BASE_URL$path")"
  if [ "$code" != "$expected" ]; then
    fail "$method $path returned HTTP $code, expected $expected"
  fi
}

api_get() {
  api_request GET "$1" "${2:-200}"
}

api_post() {
  api_request POST "$1" "${3:-200}" "$2"
}

compose_run() {
  docker compose run --rm "$@"
}

psql_exec() {
  docker compose exec -T postgres psql \
    -v ON_ERROR_STOP=1 \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    "$@"
}

seed_stage8_instruments() {
  psql_exec -v symbol="$SYMBOL" -v lock_symbol="$LOCK_SYMBOL" <<'SQL' >/dev/null
INSERT INTO market_instruments (
  exchange, symbol, base_asset, quote_asset, instrument_type, status, search_priority, synced_at
)
VALUES
  ('binance', :'symbol', regexp_replace(:'symbol', 'USDT$', ''), 'USDT', 'spot', 'active', 900, now()),
  ('binance', :'lock_symbol', regexp_replace(:'lock_symbol', 'USDT$', ''), 'USDT', 'spot', 'active', 901, now())
ON CONFLICT (exchange, symbol) DO UPDATE
   SET base_asset = EXCLUDED.base_asset,
       quote_asset = EXCLUDED.quote_asset,
       instrument_type = EXCLUDED.instrument_type,
       status = 'active',
       synced_at = EXCLUDED.synced_at,
       updated_at = now();
SQL
}

pause_existing_stage8_tasks() {
  psql_exec -c "
    UPDATE trading_tasks
       SET status = 'paused',
           locked_by = NULL,
           locked_until = NULL,
           heartbeat_at = NULL,
           updated_at = now()
     WHERE name LIKE 'stage8-smoke-%'
       AND status = 'running';
  " >/dev/null
}

assert_task_unlocked() {
  local table="$1"
  local id="$2"
  local expected_status="$3"
  case "$table" in
    data_sync_tasks|backtest_tasks|trading_tasks) ;;
    *) fail "unsupported lock assertion table $table" ;;
  esac
  local count
  count="$(psql_exec -At -v id="$id" -v expected_status="$expected_status" <<SQL | tr -d '[:space:]'
SELECT count(*)
  FROM $table
 WHERE id = :'id'
   AND status = :'expected_status'
   AND locked_by IS NULL
   AND locked_until IS NULL
   AND heartbeat_at IS NULL;
SQL
)"
  if [ "$count" != "1" ]; then
    fail "expected $table $id to be $expected_status and unlocked"
  fi
}

assert_notification_outbox_unlocked() {
  local task_id="$1"
  local count
  count="$(psql_exec -At -v task_id="$task_id" <<'SQL' | tr -d '[:space:]'
SELECT count(*)
  FROM notification_outbox
 WHERE task_id = :'task_id'
   AND status = 'delivered'
   AND locked_by IS NULL
   AND locked_until IS NULL;
SQL
)"
  if [ "$count" = "0" ]; then
    fail "expected delivered notification_outbox rows for $task_id to be unlocked"
  fi
}

force_data_sync_lock() {
  local id="$1"
  local sync_enabled="$2"
  local realtime_enabled="$3"
  psql_exec -v id="$id" -v sync_enabled="$sync_enabled" -v realtime_enabled="$realtime_enabled" <<'SQL' >/dev/null
UPDATE data_sync_tasks
   SET status = 'running',
       sync_enabled = :'sync_enabled'::boolean,
       realtime_enabled = :'realtime_enabled'::boolean,
       locked_by = 'stage8-lock-test',
       locked_until = now() + INTERVAL '5 minutes',
       heartbeat_at = now(),
       updated_at = now()
 WHERE id = :'id';
SQL
}

force_trading_lock() {
  local id="$1"
  psql_exec -v id="$id" <<'SQL' >/dev/null
UPDATE trading_tasks
   SET status = 'running',
       locked_by = 'stage8-lock-test',
       locked_until = now() + INTERVAL '5 minutes',
       heartbeat_at = now(),
       updated_at = now()
 WHERE id = :'id';
SQL
}

seed_candles() {
  psql_exec -v symbol="$SYMBOL" -v task_id="$DATA_TASK_ID" <<'SQL' >/dev/null
WITH raw AS (
  SELECT generate_series(0, 119) AS i
),
priced AS (
  SELECT
    i,
    CASE
      WHEN i < 20 THEN 100.0 - i * 0.20
      WHEN i < 70 THEN 96.0 + (i - 20) * 0.45
      ELSE 118.5 - (i - 70) * 0.35
    END AS close_price
  FROM raw
),
candles AS (
  SELECT
    i,
    close_price,
    LAG(close_price, 1, close_price) OVER (ORDER BY i) AS open_price
  FROM priced
)
INSERT INTO market_candles (
  exchange, symbol, interval, open_time, close_time,
  open, high, low, close, volume, is_closed, updated_at
)
SELECT
  'binance',
  :'symbol',
  '1m',
  TIMESTAMPTZ '2026-01-01T00:00:00Z' + i * INTERVAL '1 minute',
  TIMESTAMPTZ '2026-01-01T00:01:00Z' + i * INTERVAL '1 minute',
  open_price,
  GREATEST(open_price, close_price) + 0.08,
  LEAST(open_price, close_price) - 0.08,
  close_price,
  10 + i,
  true,
  now()
FROM candles
ON CONFLICT (exchange, symbol, interval, open_time)
DO UPDATE SET close_time = EXCLUDED.close_time,
              open = EXCLUDED.open,
              high = EXCLUDED.high,
              low = EXCLUDED.low,
              close = EXCLUDED.close,
              volume = EXCLUDED.volume,
              is_closed = EXCLUDED.is_closed,
              updated_at = now();

UPDATE data_sync_tasks
   SET status = 'running',
       sync_enabled = true,
       realtime_enabled = false,
       last_synced_open_time = TIMESTAMPTZ '2026-01-01T01:59:00Z',
       last_error = NULL,
       updated_at = now()
 WHERE id = :'task_id';

UPDATE data_sync_tasks
   SET status = 'succeeded',
       sync_enabled = false,
       finished_at = now(),
       updated_at = now()
 WHERE id = :'task_id';
SQL
}

wait_for_backtest() {
  local id="$1"
  for _ in $(seq 1 30); do
    api_get "/api/backtests/$id"
    if [ "$(json_get "$BODY_FILE" status)" = "succeeded" ]; then
      return 0
    fi
    compose_run backtest backtest --once >/dev/null
    sleep 1
  done
  fail "backtest $id did not succeed"
}

wait_for_trading_outputs() {
  local id="$1"
  local mode="$2"
  for _ in $(seq 1 30); do
    if [ "$mode" = "execute" ]; then
      api_get "/api/trading/tasks/$id/orders"
      local orders
      orders="$(json_get "$BODY_FILE" length)"
      api_get "/api/trading/tasks/$id/executions"
      local executions
      executions="$(json_get "$BODY_FILE" length)"
      if [ "$orders" -gt 0 ] && [ "$executions" -gt 0 ]; then
        return 0
      fi
    else
      api_get "/api/trading/tasks/$id/notifications"
      if [ "$(json_get "$BODY_FILE" length)" -gt 0 ]; then
        return 0
      fi
    fi
    compose_run trading trading --once >/dev/null
    sleep 1
  done
  fail "trading task $id did not produce $mode outputs"
}

wait_for_notifications_sent() {
  local id="$1"
  for _ in $(seq 1 60); do
    api_get "/api/trading/tasks/$id/notifications"
    local total sent
    total="$(json_get "$BODY_FILE" length)"
    sent="$(node - "$BODY_FILE" <<'NODE'
const fs = require("fs");
const rows = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
console.log(rows.filter((row) => row.status === "sent").length);
NODE
)"
    if [ "$total" -gt 0 ] && [ "$sent" -eq "$total" ]; then
      return 0
    fi
    compose_run notify notify --once >/dev/null
    sleep 1
  done
  fail "notifications for trading task $id did not reach sent state"
}

require_env POSTGRES_USER
require_env POSTGRES_DB
require_env BOOTSTRAP_OPERATOR_USERNAME
require_env BOOTSTRAP_OPERATOR_PASSWORD

log "compose up"
docker compose up -d --build
curl -fsS "$BASE_URL/readyz" >/dev/null

log "migration audit"
scripts/stage8-migration-audit.sh

log "login and csrf"
LOGIN_PAYLOAD="$(node - <<'NODE'
console.log(JSON.stringify({
  username: process.env.BOOTSTRAP_OPERATOR_USERNAME,
  password: process.env.BOOTSTRAP_OPERATOR_PASSWORD,
}));
NODE
)"
code="$(curl -sS -o "$BODY_FILE" -w "%{http_code}" -c "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  --data "$LOGIN_PAYLOAD" \
  "$BASE_URL/api/auth/login")"
if [ "$code" != "200" ]; then
  fail "login returned HTTP $code"
fi
if [ -z "$(csrf_token)" ]; then
  fail "login did not set csrf cookie"
fi

log "cleanup old stage8 runners"
pause_existing_stage8_tasks
seed_stage8_instruments

log "strategy registry"
api_get "/api/strategies"
node - "$BODY_FILE" <<'NODE'
const fs = require("fs");
const strategies = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
if (!strategies.some((item) => item.id === "ema-cross")) {
  console.error("ema-cross strategy missing");
  process.exit(1);
}
NODE

log "worker lease release on user stop"
api_post "/api/data/tasks" \
  "{\"exchange\":\"binance\",\"symbol\":\"$LOCK_SYMBOL\",\"interval\":\"1m\",\"startTime\":\"$START_TIME\",\"endTime\":\"$END_TIME\"}" \
  201
SYNC_LOCK_TASK_ID="$(json_get "$BODY_FILE" id)"
force_data_sync_lock "$SYNC_LOCK_TASK_ID" true false
api_post "/api/data/tasks/$SYNC_LOCK_TASK_ID/sync/stop" "{}"
assert_task_unlocked data_sync_tasks "$SYNC_LOCK_TASK_ID" paused

api_post "/api/data/tasks" \
  "{\"exchange\":\"binance\",\"symbol\":\"$LOCK_SYMBOL\",\"interval\":\"1m\",\"startTime\":\"$START_TIME\",\"endTime\":\"$END_TIME\"}" \
  201
REALTIME_LOCK_TASK_ID="$(json_get "$BODY_FILE" id)"
force_data_sync_lock "$REALTIME_LOCK_TASK_ID" false true
api_post "/api/data/tasks/$REALTIME_LOCK_TASK_ID/realtime/stop" "{}"
assert_task_unlocked data_sync_tasks "$REALTIME_LOCK_TASK_ID" paused

api_post "/api/trading/tasks" \
  "{\"name\":\"stage8-smoke-lock-release-$STAMP\",\"type\":\"paper\",\"exchange\":\"binance\",\"accountId\":\"paper-stage8\",\"symbol\":\"$LOCK_SYMBOL\",\"interval\":\"5m\",\"strategyId\":\"ema-cross\",\"strategyParams\":{\"fastPeriod\":2,\"slowPeriod\":5,\"orderSize\":0.1,\"signalMode\":\"order\"},\"intentPolicy\":{\"orderIntent\":\"execute\",\"notificationChannel\":\"default\"}}" \
  201
TRADING_LOCK_TASK_ID="$(json_get "$BODY_FILE" id)"
force_trading_lock "$TRADING_LOCK_TASK_ID"
api_post "/api/trading/tasks/$TRADING_LOCK_TASK_ID/pause" "{}"
assert_task_unlocked trading_tasks "$TRADING_LOCK_TASK_ID" paused

log "research task and candles"
api_post "/api/data/tasks" \
  "{\"exchange\":\"binance\",\"symbol\":\"$SYMBOL\",\"interval\":\"1m\",\"startTime\":\"$START_TIME\",\"endTime\":\"$END_TIME\"}" \
  201
DATA_TASK_ID="$(json_get "$BODY_FILE" id)"
seed_candles
api_get "/api/candles?exchange=binance&symbol=$SYMBOL&interval=5m&limit=20"
if [ "$(json_get "$BODY_FILE" source)" != "aggregated" ]; then
  fail "expected CandleProvider source aggregated"
fi
if [ "$(json_get "$BODY_FILE" health)" != "ok" ]; then
  fail "expected CandleProvider health ok"
fi
if [ "$(json_get "$BODY_FILE" candles.length)" -le 0 ]; then
  fail "expected aggregated candles"
fi

log "notification channel"
api_post "/api/system/notifications/channels" \
  "{\"name\":\"$CHANNEL\",\"provider\":\"webhook-demo\",\"target\":\"stage8-smoke-target\",\"enabled\":true}" \
  201

log "backtest worker"
api_post "/api/backtests" \
  "{\"name\":\"stage8-smoke-backtest-$STAMP\",\"exchange\":\"binance\",\"symbol\":\"$SYMBOL\",\"interval\":\"5m\",\"startTime\":\"$START_TIME\",\"endTime\":\"$END_TIME\",\"strategyId\":\"ema-cross\",\"strategyParams\":{\"fastPeriod\":2,\"slowPeriod\":5,\"orderSize\":0.1,\"signalMode\":\"order\"},\"initialBalance\":\"10000\",\"feeBps\":\"1\",\"slippageBps\":\"1\",\"triggerMode\":\"closed_candle\"}" \
  201
BACKTEST_ID="$(json_get "$BODY_FILE" id)"
wait_for_backtest "$BACKTEST_ID"
assert_task_unlocked backtest_tasks "$BACKTEST_ID" succeeded
api_get "/api/backtests/$BACKTEST_ID/orders"
if [ "$(json_get "$BODY_FILE" length)" -le 0 ]; then
  fail "expected backtest orders"
fi
api_get "/api/backtests/$BACKTEST_ID/intents"
if [ "$(json_get "$BODY_FILE" length)" -le 0 ]; then
  fail "expected backtest intents"
fi

log "paper trading claim fairness"
api_post "/api/trading/tasks" \
  "{\"name\":\"stage8-smoke-paper-execute-$STAMP\",\"type\":\"paper\",\"exchange\":\"binance\",\"accountId\":\"paper-stage8\",\"symbol\":\"$SYMBOL\",\"interval\":\"5m\",\"strategyId\":\"ema-cross\",\"strategyParams\":{\"fastPeriod\":2,\"slowPeriod\":5,\"orderSize\":0.1,\"signalMode\":\"order\"},\"intentPolicy\":{\"orderIntent\":\"execute\",\"notificationChannel\":\"$CHANNEL\"}}" \
  201
EXECUTE_TASK_ID="$(json_get "$BODY_FILE" id)"
api_post "/api/trading/tasks/$EXECUTE_TASK_ID/start" "{}"

api_post "/api/trading/tasks" \
  "{\"name\":\"stage8-smoke-paper-notify-$STAMP\",\"type\":\"paper\",\"exchange\":\"binance\",\"accountId\":\"paper-stage8\",\"symbol\":\"$SYMBOL\",\"interval\":\"5m\",\"strategyId\":\"ema-cross\",\"strategyParams\":{\"fastPeriod\":2,\"slowPeriod\":5,\"orderSize\":0.1,\"signalMode\":\"notification\"},\"intentPolicy\":{\"orderIntent\":\"notify\",\"notificationChannel\":\"$CHANNEL\"}}" \
  201
NOTIFY_TASK_ID="$(json_get "$BODY_FILE" id)"
api_post "/api/trading/tasks/$NOTIFY_TASK_ID/start" "{}"

wait_for_trading_outputs "$EXECUTE_TASK_ID" execute
wait_for_trading_outputs "$NOTIFY_TASK_ID" notify

api_get "/api/trading/tasks/$EXECUTE_TASK_ID"
if [ "$(json_get "$BODY_FILE" attemptCount)" -le 0 ]; then
  fail "expected paper execute task to be claimed at least once"
fi
api_get "/api/trading/tasks/$NOTIFY_TASK_ID"
if [ "$(json_get "$BODY_FILE" attemptCount)" -le 0 ]; then
  fail "expected paper notification task to be claimed at least once"
fi

api_get "/api/trading/tasks/$EXECUTE_TASK_ID/positions"
if [ "$(json_get "$BODY_FILE" length)" -le 0 ]; then
  fail "expected paper positions"
fi
api_post "/api/trading/tasks/$EXECUTE_TASK_ID/pause" "{}"
api_post "/api/trading/tasks/$NOTIFY_TASK_ID/pause" "{}"
wait_for_notifications_sent "$NOTIFY_TASK_ID"
assert_notification_outbox_unlocked "$NOTIFY_TASK_ID"

log "system health"
api_get "/api/system/health"
node - "$BODY_FILE" <<'NODE'
const fs = require("fs");
const health = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
const workers = new Set((health.services ?? health.workers ?? []).map((item) => item.name));
for (const name of ["sync-worker", "backtest-worker", "trading-worker", "notify-worker"]) {
  if (!workers.has(name)) {
    console.error(`worker ${name} missing from system health`);
    process.exit(1);
  }
}
NODE

cat <<SUMMARY

Stage 8 smoke passed
symbol=$SYMBOL
dataTask=$DATA_TASK_ID
backtest=$BACKTEST_ID
paperExecute=$EXECUTE_TASK_ID
paperNotify=$NOTIFY_TASK_ID
channel=$CHANNEL
SUMMARY
