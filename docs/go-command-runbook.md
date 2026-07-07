# Go Command Runbook

This runbook documents the current `hi` single-binary subcommands. The project is still `scaffold`; this document is an operating checklist, not a production readiness claim.

For Docker Compose operating procedures, backup/restore commands, and release
checklists, see [Production runbook](production-runbook.md).

## Scope

The Docker image runs the same binary with different commands:

| Command | Purpose | Long running | Primary state |
| --- | --- | --- | --- |
| `hi api` | API server and frontend static files | yes | PostgreSQL, HTTP server |
| `hi sync` | market data sync worker and instrument catalog sync | yes | data sync tasks, market candles |
| `hi backtest` | backtest worker | yes | backtest tasks, orders, intents |
| `hi trading` | paper/live task worker; live execution remains guarded | yes | trading tasks, orders, executions, positions |
| `hi notify` | notification outbox worker | yes | notification outbox, notification records |
| `hi migrate` | schema migration job | no | PostgreSQL migrations |

`--once` is supported by `sync`, `backtest`, `trading`, and `notify` for one claim cycle. It is used by smoke tests and manual repair checks.

## Required Environment

All subcommands that open PostgreSQL require:

```text
DATABASE_URL
```

All subcommands apply PostgreSQL pool limits before opening PostgreSQL:

```text
DB_MAX_CONNS            default 10, allowed 1..1000
DB_MIN_CONNS            default 0, must be <= DB_MAX_CONNS
DB_MAX_CONN_LIFETIME    default 1h
DB_MAX_CONN_IDLE_TIME   default 30m
```

Invalid pool settings fail before PostgreSQL opens, and startup summaries include
the active pool limits without logging `DATABASE_URL`.

All subcommands read logging configuration before opening PostgreSQL:

```text
LOG_LEVEL           debug | info | warn | error
LOG_FORMAT          text | json
LOG_CORRELATION_ID  optional run-level correlation id
```

Defaults are `LOG_LEVEL=info` and `LOG_FORMAT=text`. Invalid logging env values
fail before PostgreSQL opens, and errors name only the env key, not the invalid
value. When `LOG_CORRELATION_ID` is empty, the command generates one and attaches
it to all `slog` records as `correlation_id`.

`hi api` also reads:

```text
HTTP_ADDR
WEB_FRONTEND_DIST
AUTH_SESSION_TTL
AUTH_COOKIE_SECURE
BOOTSTRAP_OPERATOR_USERNAME
BOOTSTRAP_OPERATOR_PASSWORD
```

Worker commands read their own worker, lease, poll, retry, and limit settings:

```text
SYNC_WORKER_ID
SYNC_LEASE_TTL
SYNC_HEARTBEAT_INTERVAL
SYNC_POLL_INTERVAL
SYNC_BATCH_LIMIT
SYNC_OVERLAP_CANDLES
SYNC_DEFAULT_LOOKBACK
SYNC_FETCH_RETRIES
SYNC_RETRY_DELAY
SYNC_RETRY_BACKOFF
SYNC_MAX_RETRY_BACKOFF
MARKET_INSTRUMENT_SYNC_ENABLED
MARKET_INSTRUMENT_SYNC_ON_START
MARKET_INSTRUMENT_SYNC_INTERVAL
BACKTEST_WORKER_ID
BACKTEST_LEASE_TTL
BACKTEST_POLL_INTERVAL
BACKTEST_CANDLE_LIMIT
TRADING_WORKER_ID
TRADING_LEASE_TTL
TRADING_POLL_INTERVAL
TRADING_CANDLE_LIMIT
NOTIFY_WORKER_ID
NOTIFY_LEASE_TTL
NOTIFY_POLL_INTERVAL
NOTIFY_RETRY_DELAY
NOTIFY_MAX_RETRY_DELAY
```

Long-running worker commands can expose optional HTTP probes:

```text
SYNC_HEALTH_ADDR
BACKTEST_HEALTH_ADDR
TRADING_HEALTH_ADDR
NOTIFY_HEALTH_ADDR
```

When set to a TCP `host:port`, the corresponding worker serves `GET /livez`,
`GET /readyz`, and `GET /healthz` after the command has opened PostgreSQL and
before the long-running runner loop starts. `/livez` only proves the process is
reachable. `/readyz` and `/healthz` run a PostgreSQL ping and a lightweight
worker queue table read, then return HTTP 503 with `status=unavailable` if
either check fails. These probes are disabled by default and are not started for
`--once`. Task lease state, queue depth, exchange backoff, stale workers,
fetch-lock skips, and catalog health remain visible through `hi api` system
health.

Public market clients read:

```text
BINANCE_BASE_URLS
BINANCE_REQUEST_WEIGHT_LIMIT
BINANCE_REQUEST_WEIGHT_WINDOW
OKX_MARKET_REQUEST_LIMIT
OKX_MARKET_REQUEST_WINDOW
```

`SYNC_FETCH_RETRIES` and `SYNC_RETRY_DELAY` apply to both K-line fetches and the background instrument catalog fetch inside `hi sync`.

Invalid `duration`, `int`, and `bool` values fail before the command opens PostgreSQL. Error messages include the env name.

## Sensitive Values

Do not log these values:

```text
DATABASE_URL
BOOTSTRAP_OPERATOR_PASSWORD
ENCRYPTION_KEY
session secrets
API keys
private keys
provider tokens
provider credentials
DSNs
```

Startup summaries are limited to non-sensitive values such as worker id, poll interval, lease TTL, retry windows, static root, bind address, and public market rate limits.

## Local Start

Full stack:

```bash
cp .env.example .env
docker compose up --build -d
curl -fsS http://127.0.0.1:8080/readyz
docker compose ps
```

Individual command examples:

```bash
DATABASE_URL="$DATABASE_URL" go run ./cmd/hi migrate
DATABASE_URL="$DATABASE_URL" go run ./cmd/hi api
DATABASE_URL="$DATABASE_URL" go run ./cmd/hi sync --once
DATABASE_URL="$DATABASE_URL" go run ./cmd/hi backtest --once
DATABASE_URL="$DATABASE_URL" go run ./cmd/hi trading --once
DATABASE_URL="$DATABASE_URL" go run ./cmd/hi notify --once
```

## Stop

Local Compose:

```bash
docker compose stop api sync backtest trading notify
```

The worker runners handle parent context cancellation and release active leases. Container-level SIGTERM release is covered by:

```bash
scripts/stage8-sigterm-smoke.sh
```

## Config Smoke

Run the command config smoke before changing subcommand startup behavior:

```bash
scripts/stage8-command-config-smoke.sh
```

The smoke builds a local `hi` binary and verifies:

- missing `DATABASE_URL` fails with a clear error;
- invalid `LOG_LEVEL` / `LOG_FORMAT` / `LOG_CORRELATION_ID` fails before the database opens and does not echo the invalid value;
- invalid `DB_MAX_CONNS` / `DB_MIN_CONNS` / DB pool duration settings fail before the database opens;
- invalid duration, int, and bool values name the failing env;
- `SYNC_HEARTBEAT_INTERVAL > SYNC_LEASE_TTL` fails before the database opens;
- worker health probe addresses must be valid `host:port` values;
- public exchange rate-limit config is validated before runtime;
- unknown flags are reported without usage noise;
- error output does not leak the test DSN, password, or secret marker.

`scripts/quality-gate.sh` runs this smoke as a hard gate.

## Troubleshooting

If a command exits immediately:

1. Check the first `command failed` log line. Config errors should name the env.
2. Confirm `DATABASE_URL` is set only in the environment and is not copied into logs.
3. Run `scripts/stage8-command-config-smoke.sh` if the failure is config-related.
4. Run `docker compose logs <service>` for container startup summaries.
5. For worker lease or SIGTERM issues, run `scripts/stage8-sigterm-smoke.sh`.

Known remaining gaps:

- structured text / JSON log output, log level config, command run-level correlation IDs, API `X-Request-ID` and `traceparent` response headers, API access logs with `request_id` / `trace_id`, API-created data sync / backtest / trading / data sync repair task and trading notification `requestId` / `traceparent` fields, data sync / backtest / trading / notify worker task logs with `request_id` / `trace_id`, and notification provider outbound `X-Request-ID` / `traceparent` propagation exist, but W3C trace context is not propagated to exchange / broader external systems or across subcommands;
- worker subcommands have optional health probes with PostgreSQL and queue table readiness, but no richer readiness model for task backlog, exchange/provider availability, or stale worker diagnosis;
- backup/restore, shared environment secret management, capacity preflight, and backup timer templates are documented in `docs/production-runbook.md`, but still lack completed production drills, target-host scheduler evidence, external backup storage monitoring, target-environment load tests, and observed sizing records;
- no claim that these commands are production-safe.
