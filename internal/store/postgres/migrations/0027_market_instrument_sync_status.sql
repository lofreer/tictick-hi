CREATE TABLE IF NOT EXISTS market_instrument_sync_statuses (
  exchange text PRIMARY KEY,
  last_attempt_at timestamptz NOT NULL,
  last_success_at timestamptz,
  last_error text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT market_instrument_sync_statuses_exchange_check
    CHECK (exchange IN ('binance', 'okx')),
  CONSTRAINT market_instrument_sync_statuses_last_error_length_check
    CHECK (char_length(last_error) <= 500),
  CONSTRAINT market_instrument_sync_statuses_success_before_attempt_check
    CHECK (last_success_at IS NULL OR last_success_at <= last_attempt_at)
);

INSERT INTO market_instrument_sync_statuses (
  exchange, last_attempt_at, last_success_at, last_error, updated_at
)
SELECT exchange,
       COALESCE(max(synced_at), now()),
       max(synced_at),
       '',
       now()
  FROM market_instruments
 GROUP BY exchange
ON CONFLICT (exchange) DO NOTHING;
