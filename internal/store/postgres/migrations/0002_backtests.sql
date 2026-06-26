CREATE TABLE IF NOT EXISTS backtest_tasks (
  id text PRIMARY KEY,
  name text NOT NULL,
  exchange text NOT NULL,
  symbol text NOT NULL,
  interval text NOT NULL,
  start_time timestamptz,
  end_time timestamptz,
  strategy_id text NOT NULL,
  strategy_params jsonb NOT NULL DEFAULT '{}'::jsonb,
  initial_balance numeric(30, 12) NOT NULL,
  fee_bps numeric(30, 12) NOT NULL DEFAULT 0,
  slippage_bps numeric(30, 12) NOT NULL DEFAULT 0,
  trigger_mode text NOT NULL DEFAULT 'closed_candle',
  status text NOT NULL DEFAULT 'pending',
  locked_by text,
  locked_until timestamptz,
  heartbeat_at timestamptz,
  started_at timestamptz,
  finished_at timestamptz,
  last_error text,
  attempt_count integer NOT NULL DEFAULT 0,
  result_summary jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backtest_tasks_status
  ON backtest_tasks (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_backtest_tasks_lookup
  ON backtest_tasks (exchange, symbol, interval);

CREATE TABLE IF NOT EXISTS backtest_orders (
  id text PRIMARY KEY,
  backtest_id text NOT NULL REFERENCES backtest_tasks(id) ON DELETE CASCADE,
  intent_id text,
  side text NOT NULL,
  price numeric(30, 12) NOT NULL,
  quantity numeric(30, 12) NOT NULL,
  status text NOT NULL,
  occurred_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_backtest_orders_backtest
  ON backtest_orders (backtest_id, occurred_at);
