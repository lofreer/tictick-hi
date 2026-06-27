ALTER TABLE data_sync_tasks
  ADD CONSTRAINT data_sync_tasks_lease_consistency_check
    CHECK (
      (
        locked_by IS NULL
        AND locked_until IS NULL
        AND heartbeat_at IS NULL
      )
      OR (
        status = 'running'
        AND locked_by IS NOT NULL
        AND locked_until IS NOT NULL
        AND heartbeat_at IS NOT NULL
      )
    ) NOT VALID;

ALTER TABLE backtest_tasks
  ADD CONSTRAINT backtest_tasks_lease_consistency_check
    CHECK (
      (
        locked_by IS NULL
        AND locked_until IS NULL
        AND heartbeat_at IS NULL
      )
      OR (
        status = 'running'
        AND locked_by IS NOT NULL
        AND locked_until IS NOT NULL
        AND heartbeat_at IS NOT NULL
      )
    ) NOT VALID;

ALTER TABLE trading_tasks
  ADD CONSTRAINT trading_tasks_lease_consistency_check
    CHECK (
      (
        locked_by IS NULL
        AND locked_until IS NULL
        AND heartbeat_at IS NULL
      )
      OR (
        status = 'running'
        AND locked_by IS NOT NULL
        AND locked_until IS NOT NULL
        AND heartbeat_at IS NOT NULL
      )
    ) NOT VALID;

ALTER TABLE notification_outbox
  ADD CONSTRAINT notification_outbox_lease_consistency_check
    CHECK (
      (
        locked_by IS NULL
        AND locked_until IS NULL
      )
      OR (
        status = 'running'
        AND locked_by IS NOT NULL
        AND locked_until IS NOT NULL
      )
    ) NOT VALID;
