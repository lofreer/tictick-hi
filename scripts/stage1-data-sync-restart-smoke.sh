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
STAMP="${STAGE1_RESTART_STAMP:-$(date +%s)}"
SYMBOL="S1RESTART${STAMP}USDT"
TASK_ID="dst_s1restart_${STAMP}"
WORKER_ID="stage1-restart-${STAMP}"
BASE_TIME="2026-01-01T00:00:00Z"
CURSOR_TIME="2026-01-01T00:02:00Z"
WANT_CURSOR_TIME="2026-01-01T00:04:00Z"

TMP_DIR="$(mktemp -d)"
SERVER_JS="$TMP_DIR/restart-market.js"
COMPOSE_OVERRIDE="$TMP_DIR/docker-compose.restart.yml"
COOKIE_JAR="$TMP_DIR/cookies.txt"
BODY_FILE="$TMP_DIR/body.json"
BACKOFF_FILE="$TMP_DIR/binance-backoff.tsv"

COMPOSE_BASE=(-f "$ROOT_DIR/docker-compose.yml")
COMPOSE_ARGS=(-f "$ROOT_DIR/docker-compose.yml" -f "$COMPOSE_OVERRIDE")
API_WAS_RUNNING="false"
SYNC_WAS_RUNNING="false"
STATE_CAPTURED="false"

log() {
  printf '\n== stage1 data sync restart smoke: %s ==\n' "$1"
}

psql_exec() {
  docker compose "${COMPOSE_ARGS[@]}" exec -T postgres psql \
    -v ON_ERROR_STOP=1 \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    "$@"
}

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  if [ -n "${TASK_ID:-}" ]; then
    printf 'data sync task state:\n' >&2
    psql_exec -At -v id="$TASK_ID" <<'SQL' >&2 || true
SELECT id,
       status,
       sync_enabled,
       realtime_enabled,
       COALESCE(locked_by, ''),
       COALESCE(last_synced_open_time::text, ''),
       attempt_count,
       COALESCE(last_error, '')
  FROM data_sync_tasks
 WHERE id = :'id';
SQL
  fi
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

detect_service_state() {
  local service="$1"
  local variable="$2"
  local container_id
  container_id="$(docker compose "${COMPOSE_BASE[@]}" ps -q "$service" 2>/dev/null || true)"
  if [ -z "$container_id" ]; then
    return 0
  fi
  if [ "$(docker inspect -f '{{.State.Running}}' "$container_id" 2>/dev/null || true)" = "true" ]; then
    printf -v "$variable" "true"
  fi
}

restore_service_state() {
  local service="$1"
  local was_running="$2"
  if [ "$was_running" = "true" ]; then
    docker compose "${COMPOSE_BASE[@]}" up -d --force-recreate "$service" >/dev/null 2>&1 || true
  else
    docker compose "${COMPOSE_BASE[@]}" rm -f -s "$service" >/dev/null 2>&1 || true
  fi
}

cleanup() {
  local exit_code=$?
  if [ -f "$COMPOSE_OVERRIDE" ]; then
    docker compose "${COMPOSE_ARGS[@]}" stop sync >/dev/null 2>&1 || true
    pause_restart_task >/dev/null 2>&1 || true
    restore_binance_backoff >/dev/null 2>&1 || true
    docker compose "${COMPOSE_ARGS[@]}" rm -f -s -v restart-market >/dev/null 2>&1 || true
  fi
  if [ "$STATE_CAPTURED" = "true" ]; then
    restore_service_state api "$API_WAS_RUNNING"
    restore_service_state sync "$SYNC_WAS_RUNNING"
  fi
  rm -rf "$TMP_DIR"
  exit "$exit_code"
}
trap cleanup EXIT

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

write_restart_compose() {
  cat > "$SERVER_JS" <<'NODE'
const http = require("http");

const symbol = process.env.STAGE1_RESTART_SYMBOL;
const baseOpen = Date.parse(process.env.STAGE1_RESTART_BASE_TIME);
const delayMs = Number(process.env.STAGE1_RESTART_DELAY_MS || "2500");
let hits = 0;
let lastQuery = null;

function candle(index) {
  const openTime = baseOpen + index * 60_000;
  const price = 100 + index;
  return [
    openTime,
    price.toFixed(2),
    (price + 1).toFixed(2),
    (price - 1).toFixed(2),
    price.toFixed(2),
    (10 + index).toFixed(2),
    openTime + 59_999,
    "0",
    0,
    "0",
    "0",
    "0"
  ];
}

const server = http.createServer((request, response) => {
  if (request.url === "/ready") {
    response.writeHead(200, { "content-type": "text/plain" });
    response.end("ok");
    return;
  }

  if (request.url === "/status") {
    response.writeHead(200, { "content-type": "application/json" });
    response.end(JSON.stringify({ hits, lastQuery }));
    return;
  }

  if (request.url.startsWith("/api/v3/klines")) {
    const url = new URL(request.url, "http://127.0.0.1");
    lastQuery = Object.fromEntries(url.searchParams.entries());
    hits += 1;
    setTimeout(() => {
      if (url.searchParams.get("symbol") !== symbol || url.searchParams.get("interval") !== "1m") {
        response.writeHead(400, { "content-type": "application/json" });
        response.end(JSON.stringify({ code: -1121, msg: "Invalid symbol." }));
        return;
      }
      response.writeHead(200, { "content-type": "application/json" });
      response.end(JSON.stringify([0, 1, 2, 3, 4].map(candle)));
    }, delayMs);
    return;
  }

  response.writeHead(404, { "content-type": "text/plain" });
  response.end("not found");
});

server.listen(8080, "0.0.0.0");

process.on("SIGTERM", () => {
  server.close(() => process.exit(0));
  setTimeout(() => process.exit(0), 1000).unref();
});
NODE

  cat > "$COMPOSE_OVERRIDE" <<YAML
services:
  restart-market:
    image: node:24-bookworm-slim
    working_dir: /srv
    command: ["node", "/srv/restart-market.js"]
    environment:
      STAGE1_RESTART_SYMBOL: "$SYMBOL"
      STAGE1_RESTART_BASE_TIME: "$BASE_TIME"
      STAGE1_RESTART_DELAY_MS: "2500"
    volumes:
      - "$SERVER_JS:/srv/restart-market.js:ro"
    restart: "no"

  sync:
    environment:
      BINANCE_BASE_URLS: http://restart-market:8080
      SYNC_WORKER_ID: "$WORKER_ID"
      SYNC_LEASE_TTL: 8s
      SYNC_HEARTBEAT_INTERVAL: 1s
      SYNC_POLL_INTERVAL: 30s
      SYNC_FETCH_RETRIES: "1"
      SYNC_RETRY_DELAY: 100ms
      SYNC_BATCH_LIMIT: "10"
      MARKET_INSTRUMENT_SYNC_ENABLED: "false"
      MARKET_INSTRUMENT_SYNC_ON_START: "false"
    depends_on:
      postgres:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
      restart-market:
        condition: service_started
YAML
}

wait_for_api() {
  for _ in $(seq 1 60); do
    if curl -fsS "$BASE_URL/readyz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  fail "api did not become ready"
}

wait_for_mock() {
  for _ in $(seq 1 60); do
    if docker compose "${COMPOSE_ARGS[@]}" exec -T restart-market node - <<'NODE' >/dev/null 2>&1
fetch("http://127.0.0.1:8080/ready")
  .then((response) => process.exit(response.ok ? 0 : 1))
  .catch(() => process.exit(1));
NODE
    then
      return 0
    fi
    sleep 1
  done
  fail "restart market mock did not become ready"
}

mock_status_value() {
  local field="$1"
  docker compose "${COMPOSE_ARGS[@]}" exec -T restart-market node - "$field" <<'NODE'
const field = process.argv[2];
fetch("http://127.0.0.1:8080/status")
  .then(async (response) => {
    if (!response.ok) process.exit(1);
    const status = await response.json();
    const value = status[field];
    console.log(typeof value === "object" ? JSON.stringify(value) : String(value ?? ""));
  })
  .catch(() => process.exit(1));
NODE
}

login() {
  local payload
  payload="$(node - <<'NODE'
console.log(JSON.stringify({
  username: process.env.BOOTSTRAP_OPERATOR_USERNAME,
  password: process.env.BOOTSTRAP_OPERATOR_PASSWORD,
}));
NODE
)"
  local code
  code="$(curl -sS -o "$BODY_FILE" -w "%{http_code}" -c "$COOKIE_JAR" \
    -H "Content-Type: application/json" \
    --data "$payload" \
    "$BASE_URL/api/auth/login")"
  if [ "$code" != "200" ]; then
    fail "login returned HTTP $code"
  fi
  if [ -z "$(csrf_token)" ]; then
    fail "login did not set csrf cookie"
  fi
}

save_binance_backoff() {
  psql_exec -At -F $'\t' <<'SQL' > "$BACKOFF_FILE"
SELECT next_attempt_at, replace(COALESCE(last_error, ''), E'\t', ' ')
  FROM data_sync_exchange_backoffs
 WHERE exchange = 'binance';
SQL
}

clear_binance_backoff() {
  psql_exec <<'SQL' >/dev/null
DELETE FROM data_sync_exchange_backoffs
 WHERE exchange = 'binance';
SQL
}

restore_binance_backoff() {
  if [ -s "$BACKOFF_FILE" ]; then
    local next_attempt_at last_error
    IFS=$'\t' read -r next_attempt_at last_error < "$BACKOFF_FILE"
    psql_exec -v next_attempt_at="$next_attempt_at" -v last_error="$last_error" <<'SQL' >/dev/null
INSERT INTO data_sync_exchange_backoffs (exchange, next_attempt_at, last_error, updated_at)
VALUES ('binance', :'next_attempt_at'::timestamptz, :'last_error', now())
ON CONFLICT (exchange)
DO UPDATE SET next_attempt_at = EXCLUDED.next_attempt_at,
              last_error = EXCLUDED.last_error,
              updated_at = now();
SQL
    return 0
  fi
  psql_exec <<'SQL' >/dev/null
DELETE FROM data_sync_exchange_backoffs
 WHERE exchange = 'binance';
SQL
}

pause_existing_restart_tasks() {
  psql_exec <<'SQL' >/dev/null
UPDATE data_sync_tasks
   SET status = 'paused',
       sync_enabled = false,
       realtime_enabled = false,
       locked_by = NULL,
       locked_until = NULL,
       heartbeat_at = NULL,
       updated_at = now()
 WHERE symbol LIKE 'S1RESTART%';
SQL
}

seed_restart_state() {
  psql_exec \
    -v id="$TASK_ID" \
    -v symbol="$SYMBOL" \
    -v base_time="$BASE_TIME" \
    -v cursor_time="$CURSOR_TIME" <<'SQL' >/dev/null
INSERT INTO market_instruments (
  exchange, symbol, base_asset, quote_asset, instrument_type, status, search_priority, synced_at
)
VALUES (
  'binance', :'symbol', regexp_replace(:'symbol', 'USDT$', ''), 'USDT', 'spot', 'active', 920, now()
)
ON CONFLICT (exchange, symbol) DO UPDATE
   SET base_asset = EXCLUDED.base_asset,
       quote_asset = EXCLUDED.quote_asset,
       instrument_type = EXCLUDED.instrument_type,
       status = 'active',
       synced_at = EXCLUDED.synced_at,
       updated_at = now();

INSERT INTO data_sync_tasks (
  id, exchange, symbol, interval, start_time, sync_enabled, realtime_enabled, status,
  locked_by, locked_until, heartbeat_at, started_at, last_synced_open_time,
  attempt_count, created_at, updated_at
)
VALUES (
  :'id', 'binance', :'symbol', '1m', :'base_time'::timestamptz, false, true, 'running',
  'stage1-crashed-worker', now() - INTERVAL '5 seconds', now() - INTERVAL '1 minute',
  now() - INTERVAL '2 minutes', :'cursor_time'::timestamptz,
  1, TIMESTAMPTZ '2000-01-01T00:00:00Z', now()
)
ON CONFLICT (id) DO UPDATE
   SET exchange = EXCLUDED.exchange,
       symbol = EXCLUDED.symbol,
       interval = EXCLUDED.interval,
       start_time = EXCLUDED.start_time,
       end_time = NULL,
       sync_enabled = false,
       realtime_enabled = true,
       status = 'running',
       locked_by = 'stage1-crashed-worker',
       locked_until = now() - INTERVAL '5 seconds',
       heartbeat_at = now() - INTERVAL '1 minute',
       started_at = now() - INTERVAL '2 minutes',
       finished_at = NULL,
       last_synced_open_time = EXCLUDED.last_synced_open_time,
       last_error = NULL,
       next_attempt_at = NULL,
       attempt_count = 1,
       created_at = EXCLUDED.created_at,
       updated_at = now();

WITH raw AS (
  SELECT generate_series(0, 2) AS i
)
INSERT INTO market_candles (
  exchange, symbol, interval, open_time, close_time,
  open, high, low, close, volume, is_closed, updated_at
)
SELECT
  'binance',
  :'symbol',
  '1m',
  :'base_time'::timestamptz + i * INTERVAL '1 minute',
  :'base_time'::timestamptz + (i + 1) * INTERVAL '1 minute',
  90 + i,
  91 + i,
  89 + i,
  90 + i,
  1 + i,
  true,
  now()
FROM raw
ON CONFLICT (exchange, symbol, interval, open_time)
DO UPDATE SET close_time = EXCLUDED.close_time,
              open = EXCLUDED.open,
              high = EXCLUDED.high,
              low = EXCLUDED.low,
              close = EXCLUDED.close,
              volume = EXCLUDED.volume,
              is_closed = EXCLUDED.is_closed,
              updated_at = now();
SQL
}

pause_restart_task() {
  psql_exec -v id="$TASK_ID" <<'SQL'
UPDATE data_sync_tasks
   SET status = 'paused',
       sync_enabled = false,
       realtime_enabled = false,
       locked_by = NULL,
       locked_until = NULL,
       heartbeat_at = NULL,
       updated_at = now()
 WHERE id = :'id';
SQL
}

wait_for_reclaimed_fetch() {
  for _ in $(seq 1 30); do
    local claimed hits
    claimed="$(psql_exec -At -v id="$TASK_ID" -v worker="$WORKER_ID" <<'SQL' | tr -d '[:space:]'
SELECT count(*)
  FROM data_sync_tasks
 WHERE id = :'id'
   AND status = 'running'
   AND realtime_enabled = true
   AND locked_by = :'worker'
   AND locked_until > now()
   AND heartbeat_at IS NOT NULL
   AND attempt_count >= 2;
SQL
)"
    hits="$(mock_status_value hits 2>/dev/null || printf '0')"
    if [ "$claimed" = "1" ] && [ "${hits:-0}" -gt 0 ]; then
      return 0
    fi
    sleep 1
  done
  fail "sync container did not reclaim the expired realtime task"
}

wait_for_resume_result() {
  for _ in $(seq 1 45); do
    local task_ready candle_ready
    task_ready="$(psql_exec -At -v id="$TASK_ID" -v want="$WANT_CURSOR_TIME" <<'SQL' | tr -d '[:space:]'
SELECT count(*)
  FROM data_sync_tasks
 WHERE id = :'id'
   AND status = 'running'
   AND sync_enabled = false
   AND realtime_enabled = true
   AND locked_by IS NULL
   AND locked_until IS NULL
   AND heartbeat_at IS NULL
   AND last_synced_open_time = :'want'::timestamptz
   AND attempt_count >= 2
   AND COALESCE(last_error, '') = '';
SQL
)"
    candle_ready="$(psql_exec -At -v symbol="$SYMBOL" -v base="$BASE_TIME" <<'SQL' | tr -d '[:space:]'
WITH expected AS (
  SELECT
    :'base'::timestamptz + i * INTERVAL '1 minute' AS open_time,
    (100 + i)::numeric AS close
  FROM generate_series(0, 4) AS raw(i)
)
SELECT count(*)
  FROM market_candles candle
  JOIN expected ON expected.open_time = candle.open_time
 WHERE candle.exchange = 'binance'
   AND candle.symbol = :'symbol'
   AND candle.interval = '1m'
   AND candle.close = expected.close;
SQL
)"
    if [ "$task_ready" = "1" ] && [ "$candle_ready" = "5" ]; then
      return 0
    fi
    sleep 1
  done
  fail "sync container did not persist resumed candles and cursor"
}

assert_api_task_visible() {
  api_get "/api/data/tasks"
  node - "$BODY_FILE" "$TASK_ID" "$WANT_CURSOR_TIME" <<'NODE'
const fs = require("fs");
const rows = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
const taskID = process.argv[3];
const wantCursor = Date.parse(process.argv[4]);
const task = rows.find((row) => row.id === taskID);
if (!task) {
  console.error(`task ${taskID} is missing from /api/data/tasks`);
  process.exit(1);
}
if (task.status !== "running" || task.realtimeEnabled !== true || task.syncEnabled !== false) {
  console.error(`unexpected task state ${JSON.stringify(task)}`);
  process.exit(1);
}
if (Date.parse(task.latestSyncedAt ?? "") !== wantCursor) {
  console.error(`latestSyncedAt=${task.latestSyncedAt}, want ${process.argv[4]}`);
  process.exit(1);
}
if (task.dataHealth !== "syncing") {
  console.error(`dataHealth=${task.dataHealth}, want syncing`);
  process.exit(1);
}
NODE
}

require_env POSTGRES_USER
require_env POSTGRES_PASSWORD
require_env POSTGRES_DB
require_env BOOTSTRAP_OPERATOR_USERNAME
require_env BOOTSTRAP_OPERATOR_PASSWORD

detect_service_state api API_WAS_RUNNING
detect_service_state sync SYNC_WAS_RUNNING
STATE_CAPTURED="true"
write_restart_compose

log "compose up without sync"
docker compose "${COMPOSE_ARGS[@]}" up -d --build postgres migrate api restart-market
docker compose "${COMPOSE_ARGS[@]}" stop sync >/dev/null 2>&1 || true
docker compose "${COMPOSE_ARGS[@]}" rm -f -s sync >/dev/null 2>&1 || true
wait_for_api
wait_for_mock

log "seed expired realtime lease"
login
save_binance_backoff
clear_binance_backoff
pause_existing_restart_tasks
seed_restart_state

log "restart sync container"
docker compose "${COMPOSE_ARGS[@]}" up -d --build sync
wait_for_reclaimed_fetch

log "assert cursor and api visibility"
wait_for_resume_result
assert_api_task_visible
pause_restart_task >/dev/null

cat <<SUMMARY

Stage 1 data sync restart smoke passed
symbol=$SYMBOL
task=$TASK_ID
syncWorker=$WORKER_ID
cursor=$WANT_CURSOR_TIME
klinesHits=$(mock_status_value hits 2>/dev/null || printf 'unknown')
SUMMARY
