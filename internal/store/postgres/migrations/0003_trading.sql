CREATE TABLE IF NOT EXISTS trading_tasks (
  id text PRIMARY KEY,
  name text NOT NULL,
  type text NOT NULL,
  exchange text NOT NULL,
  account_id text NOT NULL,
  symbol text NOT NULL,
  strategy_id text NOT NULL,
  strategy_params jsonb NOT NULL DEFAULT '{}'::jsonb,
  intent_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL DEFAULT 'pending',
  locked_by text,
  locked_until timestamptz,
  heartbeat_at timestamptz,
  started_at timestamptz,
  finished_at timestamptz,
  last_error text,
  attempt_count integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trading_tasks_status
  ON trading_tasks (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_trading_tasks_lookup
  ON trading_tasks (type, exchange, account_id, symbol);

CREATE TABLE IF NOT EXISTS strategy_intents (
  id text PRIMARY KEY,
  task_id text NOT NULL,
  task_type text NOT NULL,
  strategy_id text NOT NULL,
  intent_type text NOT NULL,
  idempotency_key text NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  policy text NOT NULL,
  status text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (task_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_strategy_intents_task
  ON strategy_intents (task_id, created_at DESC);

CREATE TABLE IF NOT EXISTS orders (
  id text PRIMARY KEY,
  task_id text NOT NULL,
  task_type text NOT NULL,
  intent_id text,
  idempotency_key text NOT NULL,
  exchange text NOT NULL,
  account_id text NOT NULL,
  symbol text NOT NULL,
  side text NOT NULL,
  order_type text NOT NULL,
  price numeric(30, 12) NOT NULL,
  quantity numeric(30, 12) NOT NULL,
  status text NOT NULL,
  exchange_order_id text,
  exchange_response_summary jsonb NOT NULL DEFAULT '{}'::jsonb,
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (task_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_orders_task
  ON orders (task_id, created_at DESC);

CREATE TABLE IF NOT EXISTS notifications (
  id text PRIMARY KEY,
  task_id text NOT NULL,
  intent_id text,
  channel text NOT NULL,
  title text NOT NULL,
  body text NOT NULL,
  status text NOT NULL,
  error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  sent_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_notifications_task
  ON notifications (task_id, created_at DESC);
