ALTER TABLE data_sync_tasks
  ADD COLUMN IF NOT EXISTS next_attempt_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_data_sync_tasks_next_attempt
  ON data_sync_tasks (status, next_attempt_at, locked_until)
  WHERE sync_enabled = true OR realtime_enabled = true;
