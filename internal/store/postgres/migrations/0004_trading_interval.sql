ALTER TABLE trading_tasks
  ADD COLUMN IF NOT EXISTS interval text NOT NULL DEFAULT '1m';

CREATE INDEX IF NOT EXISTS idx_trading_tasks_market
  ON trading_tasks (exchange, symbol, interval);
