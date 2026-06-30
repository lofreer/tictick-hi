CREATE TABLE IF NOT EXISTS data_sync_exchange_fetch_lock_skips (
  exchange text PRIMARY KEY,
  skip_count bigint NOT NULL DEFAULT 0,
  last_skipped_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT data_sync_exchange_fetch_lock_skips_exchange_check
    CHECK (exchange IN ('binance', 'okx')),
  CONSTRAINT data_sync_exchange_fetch_lock_skips_count_check
    CHECK (skip_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_data_sync_exchange_fetch_lock_skips_updated
  ON data_sync_exchange_fetch_lock_skips (updated_at DESC);
