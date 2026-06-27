UPDATE data_sync_tasks
   SET locked_by = NULL,
       locked_until = NULL,
       heartbeat_at = NULL,
       updated_at = now()
 WHERE NOT (
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
     );

ALTER TABLE data_sync_tasks
  VALIDATE CONSTRAINT data_sync_tasks_lease_consistency_check;

UPDATE backtest_tasks
   SET locked_by = NULL,
       locked_until = NULL,
       heartbeat_at = NULL,
       updated_at = now()
 WHERE NOT (
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
     );

ALTER TABLE backtest_tasks
  VALIDATE CONSTRAINT backtest_tasks_lease_consistency_check;

UPDATE trading_tasks
   SET locked_by = NULL,
       locked_until = NULL,
       heartbeat_at = NULL,
       updated_at = now()
 WHERE NOT (
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
     );

ALTER TABLE trading_tasks
  VALIDATE CONSTRAINT trading_tasks_lease_consistency_check;

UPDATE notification_outbox
   SET locked_by = NULL,
       locked_until = NULL,
       updated_at = now()
 WHERE NOT (
       (
         locked_by IS NULL
         AND locked_until IS NULL
       )
       OR (
         status = 'running'
         AND locked_by IS NOT NULL
         AND locked_until IS NOT NULL
       )
     );

ALTER TABLE notification_outbox
  VALIDATE CONSTRAINT notification_outbox_lease_consistency_check;
