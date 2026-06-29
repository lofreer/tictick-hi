ALTER TABLE data_sync_tasks
  ADD COLUMN IF NOT EXISTS deleted_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_data_sync_tasks_active_lookup
  ON data_sync_tasks (exchange, symbol, interval)
  WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION enforce_data_sync_task_status_transition()
RETURNS trigger AS $$
BEGIN
  IF NEW.status = OLD.status THEN
    RETURN NEW;
  END IF;

  IF NOT (
    (NEW.status = 'cancelled' AND OLD.status <> 'cancelled')
    OR (OLD.status = 'pending' AND NEW.status IN ('running', 'paused'))
    OR (OLD.status = 'running' AND NEW.status IN ('pending', 'paused', 'succeeded', 'failed'))
    OR (OLD.status = 'paused' AND NEW.status IN ('pending', 'running'))
    OR (OLD.status = 'failed' AND NEW.status = 'pending')
    OR (OLD.status = 'succeeded' AND NEW.status IN ('pending', 'running'))
  ) THEN
    RAISE EXCEPTION 'data_sync_tasks_status_transition_check: % -> %', OLD.status, NEW.status
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
