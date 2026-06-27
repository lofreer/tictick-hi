# tictick-hi

Local full-stack runbook for the trading workspace.

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

Stop the stack:

```bash
docker compose down
```

Reset local data:

```bash
docker compose down -v
```

Change `POSTGRES_PASSWORD` and `BOOTSTRAP_OPERATOR_PASSWORD` before using the stack in any shared environment. `AUTH_COOKIE_SECURE=false` is only for local HTTP; enable it behind HTTPS.
