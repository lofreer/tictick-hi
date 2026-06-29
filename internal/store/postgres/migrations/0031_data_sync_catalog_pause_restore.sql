ALTER TABLE data_sync_tasks
  ADD COLUMN IF NOT EXISTS market_pause_sync_enabled boolean NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS market_pause_realtime_enabled boolean NOT NULL DEFAULT false;

