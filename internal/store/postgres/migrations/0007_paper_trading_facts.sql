CREATE TABLE IF NOT EXISTS executions (
  id text PRIMARY KEY,
  task_id text NOT NULL,
  task_type text NOT NULL,
  order_id text NOT NULL,
  intent_id text,
  idempotency_key text NOT NULL,
  exchange text NOT NULL,
  account_id text NOT NULL,
  symbol text NOT NULL,
  side text NOT NULL,
  price numeric(30, 12) NOT NULL,
  quantity numeric(30, 12) NOT NULL,
  fee numeric(30, 12) NOT NULL DEFAULT 0,
  status text NOT NULL,
  executed_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (task_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_executions_task
  ON executions (task_id, executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_executions_order
  ON executions (order_id);

CREATE TABLE IF NOT EXISTS positions (
  task_id text NOT NULL,
  task_type text NOT NULL,
  exchange text NOT NULL,
  account_id text NOT NULL,
  symbol text NOT NULL,
  quantity numeric(30, 12) NOT NULL,
  average_price numeric(30, 12) NOT NULL,
  realized_pnl numeric(30, 12) NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (task_id, symbol)
);

CREATE INDEX IF NOT EXISTS idx_positions_task
  ON positions (task_id, symbol);
