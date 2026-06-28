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

Run the research chart height smoke after the stack is up:

```bash
node scripts/research-chart-height-smoke.mjs
```

This launches an isolated headless Chrome, signs in to the local stack, opens `/research`, and fails if the K-line chart height grows during repeated desktop or mobile sampling.

Run the Stage 8 visual smoke after the stack is up:

```bash
node scripts/stage8-visual-smoke.mjs
```

This signs in to the local stack and checks the core pages in desktop/mobile viewports and light/dark themes for runtime errors, horizontal overflow, and missing primary content.

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
