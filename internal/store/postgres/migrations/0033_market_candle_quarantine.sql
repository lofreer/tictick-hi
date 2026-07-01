CREATE TABLE IF NOT EXISTS market_candle_quarantines (
  id bigserial PRIMARY KEY,
  exchange text NOT NULL,
  symbol text NOT NULL,
  interval text NOT NULL,
  open_time timestamptz NOT NULL,
  close_time timestamptz NOT NULL,
  open numeric(30, 12) NOT NULL,
  high numeric(30, 12) NOT NULL,
  low numeric(30, 12) NOT NULL,
  close numeric(30, 12) NOT NULL,
  volume numeric(30, 12) NOT NULL,
  is_closed boolean NOT NULL,
  reason text NOT NULL,
  message text NOT NULL DEFAULT '',
  quarantined_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (exchange, symbol, interval, open_time, reason)
);

CREATE INDEX IF NOT EXISTS idx_market_candle_quarantines_lookup
  ON market_candle_quarantines (exchange, symbol, interval, quarantined_at DESC);
