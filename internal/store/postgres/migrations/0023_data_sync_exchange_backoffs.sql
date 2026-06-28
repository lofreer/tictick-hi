CREATE TABLE IF NOT EXISTS data_sync_exchange_backoffs (
  exchange text PRIMARY KEY,
  next_attempt_at timestamptz NOT NULL,
  last_error text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_data_sync_exchange_backoffs_next_attempt
  ON data_sync_exchange_backoffs (next_attempt_at);
