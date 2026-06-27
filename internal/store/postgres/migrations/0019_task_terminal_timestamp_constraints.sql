UPDATE data_sync_tasks
   SET finished_at = COALESCE(finished_at, updated_at, now())
 WHERE status IN ('succeeded', 'failed', 'cancelled')
   AND finished_at IS NULL;

ALTER TABLE data_sync_tasks
  ADD CONSTRAINT data_sync_tasks_terminal_finished_at_check
    CHECK (
      status NOT IN ('succeeded', 'failed', 'cancelled')
      OR finished_at IS NOT NULL
    );

UPDATE backtest_tasks
   SET finished_at = COALESCE(finished_at, updated_at, now())
 WHERE status IN ('succeeded', 'failed', 'cancelled')
   AND finished_at IS NULL;

ALTER TABLE backtest_tasks
  ADD CONSTRAINT backtest_tasks_terminal_finished_at_check
    CHECK (
      status NOT IN ('succeeded', 'failed', 'cancelled')
      OR finished_at IS NOT NULL
    );

UPDATE trading_tasks
   SET finished_at = COALESCE(finished_at, updated_at, now())
 WHERE status IN ('succeeded', 'failed', 'cancelled')
   AND finished_at IS NULL;

ALTER TABLE trading_tasks
  ADD CONSTRAINT trading_tasks_terminal_finished_at_check
    CHECK (
      status NOT IN ('succeeded', 'failed', 'cancelled')
      OR finished_at IS NOT NULL
    );
