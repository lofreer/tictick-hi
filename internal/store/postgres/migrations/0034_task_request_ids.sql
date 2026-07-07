ALTER TABLE data_sync_tasks
  ADD COLUMN IF NOT EXISTS request_id text;

ALTER TABLE backtest_tasks
  ADD COLUMN IF NOT EXISTS request_id text;

ALTER TABLE trading_tasks
  ADD COLUMN IF NOT EXISTS request_id text;

CREATE INDEX IF NOT EXISTS idx_data_sync_tasks_request_id
  ON data_sync_tasks (request_id)
  WHERE request_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_backtest_tasks_request_id
  ON backtest_tasks (request_id)
  WHERE request_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trading_tasks_request_id
  ON trading_tasks (request_id)
  WHERE request_id IS NOT NULL;
