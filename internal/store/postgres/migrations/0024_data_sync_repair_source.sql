ALTER TABLE data_sync_tasks
  ADD COLUMN IF NOT EXISTS repair_source_task_id text;

DO $$
BEGIN
  ALTER TABLE data_sync_tasks
    ADD CONSTRAINT data_sync_tasks_repair_source_not_self_check
    CHECK (repair_source_task_id IS NULL OR repair_source_task_id <> id);
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TABLE data_sync_tasks
    ADD CONSTRAINT data_sync_tasks_repair_source_fk
    FOREIGN KEY (repair_source_task_id)
    REFERENCES data_sync_tasks(id)
    ON DELETE SET NULL;
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_data_sync_tasks_repair_source
  ON data_sync_tasks (repair_source_task_id)
  WHERE repair_source_task_id IS NOT NULL;
