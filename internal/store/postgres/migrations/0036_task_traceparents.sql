ALTER TABLE data_sync_tasks
  ADD COLUMN IF NOT EXISTS traceparent text;

ALTER TABLE backtest_tasks
  ADD COLUMN IF NOT EXISTS traceparent text;

ALTER TABLE trading_tasks
  ADD COLUMN IF NOT EXISTS traceparent text;

CREATE INDEX IF NOT EXISTS idx_data_sync_tasks_traceparent
  ON data_sync_tasks (traceparent)
  WHERE traceparent IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_backtest_tasks_traceparent
  ON backtest_tasks (traceparent)
  WHERE traceparent IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trading_tasks_traceparent
  ON trading_tasks (traceparent)
  WHERE traceparent IS NOT NULL;
