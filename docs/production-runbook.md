# Production Runbook

This runbook documents the current Docker Compose operating boundary for
`tictick-hi`. The project remains `scaffold`; this document is not a
production-safety claim.

Use this runbook together with:

- [Go command runbook](go-command-runbook.md)
- [Quality audit](quality-audit.md)
- [AI delivery protocol](ai-delivery-protocol.md)

## Scope

Current first-version topology:

| Service | Command | Role | Long running |
| --- | --- | --- | --- |
| `postgres` | PostgreSQL | database and coordination center | yes |
| `migrate` | `hi migrate` | schema migration job | no |
| `api` | `hi api` | API server and frontend static files | yes |
| `sync` | `hi sync` | data sync and instrument catalog worker | yes |
| `backtest` | `hi backtest` | backtest worker | yes |
| `trading` | `hi trading` | paper/live task worker; live remains guarded | yes |
| `notify` | `hi notify` | notification outbox worker | yes |

The only stateful service in the Compose file is PostgreSQL. Application
containers must be replaceable because runtime state belongs in PostgreSQL or in
explicit external systems.

## Required Decisions Before Shared Use

Do not use `.env.example` values in a shared environment.

Before starting a shared stack, decide and record:

- external host, TLS termination, and `HTTP_PORT`;
- `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB`;
- backup location, retention, restore owner, and restore drill schedule;
- `ENCRYPTION_KEY` source and rotation owner;
- `BOOTSTRAP_OPERATOR_USERNAME` and first password handoff;
- `AUTH_COOKIE_SECURE=true` when the app is served behind HTTPS;
- Binance / OKX public market base URLs and rate-limit settings;
- notification provider secret source for Telegram, Feishu, and SMTP;
- who can edit `.env`, restart containers, restore backups, and read logs.

Do not commit `.env`, database dumps, exchange keys, notification tokens, or
provider credentials.

## First Start

Prepare environment values:

```bash
cp .env.example .env
$EDITOR .env
```

Build and start:

```bash
docker compose up --build -d
docker compose ps
curl -fsS http://127.0.0.1:${HTTP_PORT:-8080}/readyz
```

Check startup logs without copying secrets into tickets or chat:

```bash
docker compose logs --tail=120 migrate
docker compose logs --tail=120 api
docker compose logs --tail=120 sync
docker compose logs --tail=120 backtest
docker compose logs --tail=120 trading
docker compose logs --tail=120 notify
```

The expected steady state is:

- `migrate` exited successfully;
- `api` is healthy;
- long-running workers are running or intentionally scaled to zero;
- `/readyz` returns success;
- the system health page can load after login.

## Routine Health Checks

API readiness:

```bash
curl -fsS http://127.0.0.1:${HTTP_PORT:-8080}/readyz
```

Container state:

```bash
docker compose ps
```

Database readiness:

```bash
docker compose exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"
```

Database pool limits:

```bash
DB_MAX_CONNS=10
DB_MIN_CONNS=0
DB_MAX_CONN_LIFETIME=1h
DB_MAX_CONN_IDLE_TIME=30m
```

These settings apply to every `hi` process before PostgreSQL opens. Keep
`DB_MAX_CONNS * number_of_hi_processes` below the PostgreSQL capacity reserved
for the application. Invalid pool settings stop commands before PostgreSQL opens
and do not log `DATABASE_URL`.

Capacity preflight:

```bash
scripts/stage8-capacity-check.sh
```

The check validates declared `hi` process count, PostgreSQL connection budget,
CPU / memory budget, free disk, estimated daily backup size, and retention days.
Override `STAGE8_HI_PROCESS_COUNT`, `STAGE8_POSTGRES_MAX_CONNECTIONS`,
`STAGE8_POSTGRES_RESERVED_CONNECTIONS`, `STAGE8_CAPACITY_CPU_MILLICORES`,
`STAGE8_CAPACITY_MEMORY_MB`, `STAGE8_CAPACITY_PATH`,
`STAGE8_BACKUP_DAILY_ESTIMATE_MB`, and `STAGE8_BACKUP_RETENTION_DAYS` for the
target environment before a release. This is a deterministic budget check, not a
substitute for load testing or observed production sizing.

Logging:

```bash
LOG_LEVEL=info
LOG_FORMAT=text
LOG_CORRELATION_ID=
```

Use `LOG_FORMAT=json` when logs are collected by a structured log pipeline.
When `LOG_CORRELATION_ID` is empty, each command process generates one and
attaches it to every `slog` record as `correlation_id`. Invalid logging settings
stop commands before PostgreSQL opens and do not echo the invalid value.

`hi api` accepts valid `X-Request-ID` and W3C `traceparent` headers and returns
both headers on responses. Missing or invalid values are replaced and invalid
input is not echoed. HTTP access logs include `request_id`, `trace_id`, `method`,
path without query string, status, bytes, and duration. API-created data sync,
backtest, trading, data sync repair tasks, and trading-task notifications persist
`X-Request-ID` as `requestId`; API-created data sync, backtest, trading, data
sync repair tasks, and trading-task notifications also persist W3C
`traceparent`. Data sync, backtest, trading, and notify worker task logs include
`request_id` / `trace_id` when the claimed task or delivery has them.
Notification provider outbound HTTP requests and SMTP message headers carry
`X-Request-ID` and `traceparent` when the delivery has them. This is still
partial correlation: W3C trace context is not yet propagated to exchange /
broader external systems or subcommands.

Optional worker process probes:

```bash
SYNC_HEALTH_ADDR=0.0.0.0:8091
BACKTEST_HEALTH_ADDR=0.0.0.0:8092
TRADING_HEALTH_ADDR=0.0.0.0:8093
NOTIFY_HEALTH_ADDR=0.0.0.0:8094
# Optional; blank disables the backlog readiness check.
SYNC_READY_MAX_BACKLOG=
SYNC_READY_MAX_AGE=
BACKTEST_READY_MAX_BACKLOG=
BACKTEST_READY_MAX_AGE=
TRADING_READY_MAX_BACKLOG=
TRADING_READY_MAX_AGE=
NOTIFY_READY_MAX_BACKLOG=
NOTIFY_READY_MAX_AGE=
```

When these values are set before starting Compose, the corresponding worker
serves `/livez`, `/readyz`, and `/healthz` after PostgreSQL is open and before
the runner loop starts. `/livez` only proves process reachability. `/readyz` and
`/healthz` run a PostgreSQL ping plus a lightweight read of the worker's queue
table and return HTTP 503 with `status=unavailable` if either check fails. If a
positive `<COMMAND>_READY_MAX_BACKLOG` or `<COMMAND>_READY_MAX_AGE` is set, the
same endpoints also run `queue_backlog` readiness against claim-ready work. Keep
using `/system/health` for task leases, general queue depth, stale workers,
exchange backoff, fetch-lock skips, and instrument catalog status.

Operational UI checks:

- sign in to `/overview`;
- open `/system/health`;
- inspect stale workers, exchange backoff, fetch-lock skips, and catalog status;
- open `/research` for the market being synchronized and confirm data health.

## Backup

The current Compose file stores PostgreSQL data in the `postgres_data` volume.
Backups must be database dumps, not container filesystem copies.

Create a backup directory outside the repository or set `STAGE8_BACKUP_DIR`:

```bash
mkdir -p ../tictick-hi-backups
```

Validate backup configuration without calling Docker:

```bash
POSTGRES_USER="$POSTGRES_USER" POSTGRES_DB="$POSTGRES_DB" \
  scripts/stage8-backup.sh --dry-run
```

Take a compressed PostgreSQL dump and prune old `tictick-hi-*.dump` files after
`STAGE8_BACKUP_RETENTION_DAYS`:

```bash
STAGE8_BACKUP_DIR=../tictick-hi-backups \
STAGE8_BACKUP_RETENTION_DAYS=14 \
  scripts/stage8-backup.sh
```

Record the artifact name, source commit, image tag, database name, and restore
test target. Store the dump in the agreed external backup location.

Example systemd units are available in `deploy/systemd/tictick-hi-backup.service`
and `deploy/systemd/tictick-hi-backup.timer`. Copy them to the target host,
adjust `WorkingDirectory`, `EnvironmentFile`, `STAGE8_BACKUP_DIR`, and
`STAGE8_BACKUP_RETENTION_DAYS`, then run:

```bash
systemctl enable --now tictick-hi-backup.timer
systemctl list-timers tictick-hi-backup.timer
```

Current gap: the repository now contains a backup script and a systemd timer
template, but target hosts still need installation, external storage, monitoring,
and restore-drill evidence.

## Restore Drill

Run restore drills against an isolated target database. Do not restore directly
over the active database.

Automated local drill:

```bash
scripts/stage8-backup-restore-drill.sh
```

The script starts PostgreSQL if needed, runs migrations on the source database,
creates a compressed `pg_dump`, restores it into a temporary drill database,
reruns `hi migrate` against the drill database to verify migration idempotence,
checks restored table and migration metadata, and drops the drill database on
exit.

Keep the drill database for manual inspection only in a local environment:

```bash
STAGE8_BACKUP_RESTORE_KEEP_DB=1 scripts/stage8-backup-restore-drill.sh
```

Example local drill:

```bash
docker compose stop api sync backtest trading notify
docker compose exec -T postgres createdb -U "$POSTGRES_USER" tictick_hi_restore_drill
docker compose exec -T postgres pg_restore \
  -U "$POSTGRES_USER" \
  -d tictick_hi_restore_drill \
  --clean \
  --if-exists \
  < ../tictick-hi-backups/example.dump
```

After restore, point a temporary stack or temporary `DATABASE_URL` at the drill
database and run:

```bash
DATABASE_URL="postgresql://USER:PASSWORD@HOST:PORT/tictick_hi_restore_drill?sslmode=disable" \
  go run ./cmd/hi migrate

DATABASE_URL="postgresql://USER:PASSWORD@HOST:PORT/tictick_hi_restore_drill?sslmode=disable" \
  go test ./internal/store/postgres -run Integration -count=1
```

Drop the drill database only after validation results are recorded:

```bash
docker compose exec -T postgres dropdb -U "$POSTGRES_USER" tictick_hi_restore_drill
```

Current gap: the repository now has a repeatable restore drill script and backup
timer template, but recovery readiness still requires completed drill evidence
from the target environment and confirmation that the backup timer writes to the
agreed external storage.

## Upgrade

Before replacing running application containers:

1. Confirm a recent backup exists and has a recorded restore target.
2. Review migrations in `internal/store/postgres/migrations`.
3. Run the normal quality gate for the change scope.
4. Pull or build the target image.
5. Start `migrate` and verify it exits successfully.
6. Replace `api` first, then workers.

Local Compose sequence:

```bash
docker compose pull || true
docker compose build
docker compose up -d migrate
docker compose up -d api
curl -fsS http://127.0.0.1:${HTTP_PORT:-8080}/readyz
docker compose up -d sync backtest trading notify
docker compose ps
```

Migration rollback is not currently implemented as automated down migrations.
Rollback means restoring a verified backup into a compatible target database and
starting a compatible image.

## Stop And Restart

Graceful stop:

```bash
docker compose stop api sync backtest trading notify
```

Full restart without deleting data:

```bash
docker compose down
docker compose up -d
```

Delete local data only when explicitly resetting a non-shared environment:

```bash
docker compose down -v
```

Worker SIGTERM lease release is covered by `scripts/stage8-sigterm-smoke.sh`,
but that smoke is not a long-running production proof.

## Incident Checklist

When the UI reports degraded health:

1. Open `/system/health` and preserve the affected service names and details.
2. Run `docker compose ps`.
3. Inspect logs for the affected service with `docker compose logs --tail=200`.
4. Check PostgreSQL readiness with `pg_isready`.
5. For data-sync issues, inspect active exchange backoff, stale locks, fetch-lock
   skip counters, task `dataHealth`, and market catalog status.
6. For notification issues, check the outbox status and provider env references.
7. Avoid rerunning tasks manually until the current lease, retry, or backoff
   state is understood.

If a worker is wedged and the lease is already expired, restart only the
affected worker first:

```bash
docker compose restart sync
```

Escalate to full stack restart only after preserving logs and health details.

## Validation Matrix

Run before changing Compose, command startup, migration, or worker shutdown
behavior:

```bash
go test ./...
go vet ./...
pnpm --dir web/frontend run test
pnpm --dir web/frontend run build
scripts/quality-gate.sh
scripts/stage8-command-config-smoke.sh
```

Run for release-like Compose validation:

```bash
scripts/stage8-smoke.sh
scripts/stage8-sigterm-smoke.sh
scripts/stage8-backup-restore-drill.sh
```

Run Stage 1 data recovery smokes when changing data sync, CandleProvider,
catalog, exchange backoff, or repair behavior:

```bash
scripts/stage1-data-sync-restart-smoke.sh
scripts/stage1-data-sync-external-recovery-smoke.sh
scripts/stage1-real-exchange-data-sync-smoke.sh
scripts/stage1-candle-provider-perf-smoke.sh
```

Record skipped checks and the reason. A skipped Docker, browser, restore, or
real-exchange check lowers validation strength.

## Remaining Gaps

This runbook closes only the missing documentation entry point. It does not
close these production-safety gaps:

- backup script and systemd timer template exist, but no target-host installation, external storage monitor, or scheduler run evidence;
- no completed restore drill evidence for the target environment;
- capacity preflight exists, but no completed target-environment load test, observed sizing record, or automated retention enforcement;
- no exchange / broader external system W3C trace propagation, external log sink, or retention policy;
- no richer worker claim-success / external dependency readiness beyond PostgreSQL, queue-table-ready, and configured claim-ready backlog worker probes;
- no external uptime monitor or alert routing;
- no KMS / secret manager integration or `ENCRYPTION_KEY` rotation workflow;
- no long-running multi-instance exchange quota proof;
- no live trading production execution proof.
