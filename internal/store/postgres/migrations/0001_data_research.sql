CREATE TABLE IF NOT EXISTS data_sync_tasks (
  id text PRIMARY KEY,
  exchange text NOT NULL,
  symbol text NOT NULL,
  interval text NOT NULL,
  start_time timestamptz,
  end_time timestamptz,
  sync_enabled boolean NOT NULL DEFAULT false,
  realtime_enabled boolean NOT NULL DEFAULT false,
  status text NOT NULL DEFAULT 'pending',
  locked_by text,
  locked_until timestamptz,
  heartbeat_at timestamptz,
  started_at timestamptz,
  finished_at timestamptz,
  last_synced_open_time timestamptz,
  last_error text,
  attempt_count integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_data_sync_tasks_lookup
  ON data_sync_tasks (exchange, symbol, interval);

CREATE INDEX IF NOT EXISTS idx_data_sync_tasks_lease
  ON data_sync_tasks (status, locked_until);

CREATE TABLE IF NOT EXISTS market_candles (
  exchange text NOT NULL,
  symbol text NOT NULL,
  interval text NOT NULL,
  open_time timestamptz NOT NULL,
  close_time timestamptz NOT NULL,
  open numeric(30, 12) NOT NULL,
  high numeric(30, 12) NOT NULL,
  low numeric(30, 12) NOT NULL,
  close numeric(30, 12) NOT NULL,
  volume numeric(30, 12) NOT NULL,
  is_closed boolean NOT NULL DEFAULT true,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (exchange, symbol, interval, open_time)
);

