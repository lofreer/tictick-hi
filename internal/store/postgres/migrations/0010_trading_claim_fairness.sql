CREATE INDEX IF NOT EXISTS idx_trading_tasks_running_claim
  ON trading_tasks (status, updated_at ASC, created_at ASC)
  WHERE status = 'running';
