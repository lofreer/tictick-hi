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
STAMP="${STAGE8_SIGTERM_STAMP:-$(date +%s)}"
SYMBOL="S8TERM${STAMP}USDT"
WORKER_ID="stage8-sigterm-${STAMP}"
START_TIME="2026-01-01T00:00:00Z"
END_TIME="2026-01-01T02:00:00Z"

TMP_DIR="$(mktemp -d)"
SERVER_JS="$TMP_DIR/slow-market.js"
COMPOSE_OVERRIDE="$TMP_DIR/docker-compose.sigterm.yml"
COOKIE_JAR="$TMP_DIR/cookies.txt"
BODY_FILE="$TMP_DIR/body.json"
TASK_ID=""

COMPOSE_ARGS=(-f "$ROOT_DIR/docker-compose.yml" -f "$COMPOSE_OVERRIDE")

cleanup() {
  local exit_code=$?
  if [ -f "$COMPOSE_OVERRIDE" ]; then
    docker compose "${COMPOSE_ARGS[@]}" rm -f -s -v sigterm-market >/dev/null 2>&1 || true
  fi
  rm -rf "$TMP_DIR"
  exit "$exit_code"
}
trap cleanup EXIT

log() {
  printf '\n== stage8 sigterm smoke: %s ==\n' "$1"
}

psql_exec() {
  docker compose exec -T postgres psql \
    -v ON_ERROR_STOP=1 \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    "$@"
}

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  if [ -n "$TASK_ID" ]; then
    printf 'data sync task state:\n' >&2
    psql_exec -At -v id="$TASK_ID" <<'SQL' >&2 || true
SELECT id,
       status,
       sync_enabled,
       realtime_enabled,
       COALESCE(locked_by, ''),
       locked_until IS NULL,
       heartbeat_at IS NULL,
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

api_post() {
  api_request POST "$1" "${3:-200}" "$2"
}

write_sigterm_compose() {
  cat > "$SERVER_JS" <<'NODE'
const http = require("http");

let hits = 0;
const pending = new Set();

const server = http.createServer((request, response) => {
  if (request.url === "/ready") {
    response.writeHead(200, { "content-type": "text/plain" });
    response.end("ok");
    return;
  }

  if (request.url === "/status") {
    response.writeHead(200, { "content-type": "application/json" });
    response.end(JSON.stringify({ hits, pending: pending.size }));
    return;
  }

  if (request.url.startsWith("/api/v3/klines")) {
    hits += 1;
    pending.add(response);
    response.on("close", () => pending.delete(response));
    setTimeout(() => {
      if (response.destroyed || response.writableEnded) {
        pending.delete(response);
        return;
      }
      response.writeHead(200, { "content-type": "application/json" });
      response.end("[]");
      pending.delete(response);
    }, 60000);
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
  sigterm-market:
    image: node:24-bookworm-slim
    working_dir: /srv
    command: ["node", "/srv/slow-market.js"]
    volumes:
      - "$SERVER_JS:/srv/slow-market.js:ro"
    restart: "no"

  sync:
    environment:
      BINANCE_BASE_URLS: http://sigterm-market:8080
      SYNC_POLL_INTERVAL: 1s
      SYNC_HEARTBEAT_INTERVAL: 1s
      SYNC_FETCH_RETRIES: "1"
      SYNC_RETRY_DELAY: 100ms
      SYNC_WORKER_ID: "$WORKER_ID"
    depends_on:
      postgres:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
      sigterm-market:
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
    if docker compose "${COMPOSE_ARGS[@]}" exec -T sigterm-market node - <<'NODE' >/dev/null 2>&1
fetch("http://127.0.0.1:8080/ready")
  .then((response) => process.exit(response.ok ? 0 : 1))
  .catch(() => process.exit(1));
NODE
    then
      return 0
    fi
    sleep 1
  done
  fail "sigterm market mock did not become ready"
}

mock_status_value() {
  local field="$1"
  docker compose "${COMPOSE_ARGS[@]}" exec -T sigterm-market node - "$field" <<'NODE'
const field = process.argv[2];
fetch("http://127.0.0.1:8080/status")
  .then(async (response) => {
    if (!response.ok) process.exit(1);
    const status = await response.json();
    console.log(String(status[field] ?? ""));
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

wait_for_claimed_fetch() {
  for _ in $(seq 1 60); do
    local claimed hits pending
    claimed="$(psql_exec -At -v id="$TASK_ID" -v worker="$WORKER_ID" <<'SQL' | tr -d '[:space:]'
SELECT count(*)
  FROM data_sync_tasks
 WHERE id = :'id'
   AND status = 'running'
   AND sync_enabled = false
   AND realtime_enabled = true
   AND locked_by = :'worker'
   AND locked_until > now()
   AND heartbeat_at IS NOT NULL
   AND attempt_count > 0;
SQL
)"
    hits="$(mock_status_value hits 2>/dev/null || printf '0')"
    pending="$(mock_status_value pending 2>/dev/null || printf '0')"
    if [ "$claimed" = "1" ] && [ "${hits:-0}" -gt 0 ] && [ "${pending:-0}" -gt 0 ]; then
      return 0
    fi
    sleep 1
  done
  fail "sync worker did not claim the task and enter the slow fetch"
}

assert_sigterm_release() {
  for _ in $(seq 1 30); do
    local released
    released="$(psql_exec -At -v id="$TASK_ID" <<'SQL' | tr -d '[:space:]'
SELECT count(*)
  FROM data_sync_tasks
 WHERE id = :'id'
   AND status = 'running'
   AND sync_enabled = false
   AND realtime_enabled = true
   AND locked_by IS NULL
   AND locked_until IS NULL
   AND heartbeat_at IS NULL
   AND finished_at IS NULL
   AND COALESCE(last_error, '') = ''
   AND attempt_count > 0;
SQL
)"
    if [ "$released" = "1" ]; then
      return 0
    fi
    sleep 1
  done
  fail "sync task lease was not released after container SIGTERM"
}

pause_task_after_proof() {
  psql_exec -v id="$TASK_ID" <<'SQL' >/dev/null
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

pause_existing_sigterm_tasks() {
  psql_exec <<'SQL' >/dev/null
UPDATE data_sync_tasks
   SET status = 'paused',
       sync_enabled = false,
       realtime_enabled = false,
       locked_by = NULL,
       locked_until = NULL,
       heartbeat_at = NULL,
       updated_at = now()
 WHERE symbol LIKE 'S8TERM%';
SQL
}

prioritize_controlled_task() {
  psql_exec -v id="$TASK_ID" <<'SQL' >/dev/null
UPDATE data_sync_tasks
   SET created_at = TIMESTAMPTZ '2000-01-01T00:00:00Z',
       updated_at = now()
 WHERE id = :'id';
SQL
}

require_env POSTGRES_USER
require_env POSTGRES_PASSWORD
require_env POSTGRES_DB
require_env BOOTSTRAP_OPERATOR_USERNAME
require_env BOOTSTRAP_OPERATOR_PASSWORD

write_sigterm_compose

log "compose up without sync"
docker compose "${COMPOSE_ARGS[@]}" up -d --build postgres migrate api sigterm-market
docker compose "${COMPOSE_ARGS[@]}" stop sync >/dev/null 2>&1 || true
docker compose "${COMPOSE_ARGS[@]}" rm -f -s sync >/dev/null 2>&1 || true
wait_for_api
wait_for_mock

log "login and create controlled sync task"
login
pause_existing_sigterm_tasks
api_post "/api/data/tasks" \
  "{\"exchange\":\"binance\",\"symbol\":\"$SYMBOL\",\"interval\":\"1m\",\"startTime\":\"$START_TIME\",\"endTime\":\"$END_TIME\"}" \
  201
TASK_ID="$(json_get "$BODY_FILE" id)"
api_post "/api/data/tasks/$TASK_ID/realtime/start" "{}"
prioritize_controlled_task

log "start sync with slow market endpoint"
docker compose "${COMPOSE_ARGS[@]}" up -d --build sync
wait_for_claimed_fetch

log "stop sync container with SIGTERM"
docker compose "${COMPOSE_ARGS[@]}" stop -t 10 sync >/dev/null
assert_sigterm_release
pause_task_after_proof

cat <<SUMMARY

Stage 8 SIGTERM smoke passed
symbol=$SYMBOL
dataTask=$TASK_ID
worker=$WORKER_ID
SUMMARY
