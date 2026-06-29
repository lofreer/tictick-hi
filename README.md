# tictick-hi

Local full-stack runbook for the trading workspace.

## Delivery Discipline

Current project status:

```text
scaffold
```

This project must not be described as demo-ready, usable, production-safe, or complete until the corresponding audit items are closed.

Required reading before any implementation work:

- [AI delivery protocol](docs/ai-delivery-protocol.md)
- [Quality audit](docs/quality-audit.md)
- [Implementation plan](docs/implementation-plan.md)

Lightweight quality gate:

```bash
scripts/quality-gate.sh
```

The quality gate blocks current-stage engineering regressions. It can also print non-blocking audit findings for later-stage scaffold debt; those findings still keep the overall project at `scaffold`.

The gate includes a non-browser research chart layout contract. The runtime browser smoke below is still required after chart or `/research` layout changes because it validates the built app in headless Chrome.

Full local quality gate:

```bash
scripts/full-quality-gate.sh
```

This runs the protocol's common checks in one repeatable entrypoint: Go tests, Go vet, frontend typecheck, frontend tests, frontend production build, and the lightweight quality gate. Add the heavier Docker / browser checks explicitly when validating Stage 8 behavior:

```bash
FULL_QUALITY_STAGE8=1 FULL_QUALITY_SIGTERM=1 scripts/full-quality-gate.sh
```

The same default full gate runs in GitHub Actions on pull requests, pushes to `main`, and manual dispatches.

Heavy Stage 8 smoke checks are isolated in a separate GitHub Actions workflow. They can be run manually or by the weekly schedule:

```bash
gh workflow run "Stage 8 Heavy Smoke" -f full_chain=true -f sigterm=true
```

## Local Docker

Create local environment values:

```bash
cp .env.example .env
```

Start the full stack:

```bash
docker compose up --build -d
```

Open:

```text
http://127.0.0.1:8080
```

The default local operator comes from `.env`:

```text
username: admin
password: tictick-local-admin-password
```

The Compose stack runs:

- `postgres`: persistent local PostgreSQL database.
- `migrate`: one-shot schema migration job.
- `api`: Go API plus built frontend static files.
- `sync`: market data sync worker.
- `backtest`: strategy backtest worker.
- `trading`: paper/live trading task worker.

Check runtime health:

```bash
curl -fsS http://127.0.0.1:8080/readyz
docker compose ps
```

Go subcommand runbook:

```text
docs/go-command-runbook.md
```

Run the command config smoke before changing `hi api/sync/backtest/trading/notify` startup behavior:

```bash
scripts/stage8-command-config-smoke.sh
```

Run the research chart height smoke after the stack is up:

```bash
node scripts/research-chart-height-smoke.mjs
```

This launches an isolated headless Chrome, signs in to the local stack, opens `/research`, and fails if the K-line chart height grows during repeated desktop or mobile sampling.

`scripts/stage8-smoke.sh` runs the Stage 8 browser visual smokes by default after the full-chain data is created. Use `STAGE8_BROWSER_SMOKE=0 scripts/stage8-smoke.sh` only on hosts without Chrome, and report that skip because it lowers the validation strength.

Run the Stage 8 visual smokes directly after the stack is up:

```bash
node scripts/stage8-visual-smoke.mjs
node scripts/stage8-state-visual-smoke.mjs
```

These sign in to the local stack and check the core pages, details pages, and empty/error states in desktop/mobile viewports, light/dark themes, and supported locales for runtime errors, horizontal overflow, and missing primary content.

Stop the stack:

```bash
docker compose down
```

Reset local data:

```bash
docker compose down -v
```

Change `POSTGRES_PASSWORD` and `BOOTSTRAP_OPERATOR_PASSWORD` before using the stack in any shared environment. `AUTH_COOKIE_SECURE=false` is only for local HTTP; enable it behind HTTPS.

If Docker cannot reach `api.binance.com`, set `BINANCE_BASE_URLS` in `.env`. The local default tries `https://api.binance.com` first and falls back to `https://data-api.binance.vision`.
