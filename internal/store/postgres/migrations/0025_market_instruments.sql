CREATE TABLE IF NOT EXISTS market_instruments (
  exchange text NOT NULL,
  symbol text NOT NULL,
  base_asset text NOT NULL,
  quote_asset text NOT NULL,
  instrument_type text NOT NULL DEFAULT 'spot',
  status text NOT NULL DEFAULT 'active',
  search_priority integer NOT NULL DEFAULT 100,
  synced_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (exchange, symbol)
);

CREATE INDEX IF NOT EXISTS idx_market_instruments_search
  ON market_instruments (exchange, status, search_priority, symbol);

ALTER TABLE market_instruments
  ADD CONSTRAINT market_instruments_exchange_check
    CHECK (exchange IN ('binance', 'okx')),
  ADD CONSTRAINT market_instruments_required_text_check
    CHECK (symbol <> '' AND base_asset <> '' AND quote_asset <> '' AND instrument_type <> '' AND status <> ''),
  ADD CONSTRAINT market_instruments_status_check
    CHECK (status IN ('active', 'inactive')),
  ADD CONSTRAINT market_instruments_search_priority_check
    CHECK (search_priority >= 0);

INSERT INTO market_instruments (
  exchange, symbol, base_asset, quote_asset, instrument_type, status, search_priority, synced_at
)
VALUES
  ('binance', 'BTCUSDT', 'BTC', 'USDT', 'spot', 'active', 1, now()),
  ('binance', 'ETHUSDT', 'ETH', 'USDT', 'spot', 'active', 2, now()),
  ('binance', 'SOLUSDT', 'SOL', 'USDT', 'spot', 'active', 3, now()),
  ('binance', 'BNBUSDT', 'BNB', 'USDT', 'spot', 'active', 4, now()),
  ('binance', 'XRPUSDT', 'XRP', 'USDT', 'spot', 'active', 5, now()),
  ('binance', 'ADAUSDT', 'ADA', 'USDT', 'spot', 'active', 6, now()),
  ('binance', 'DOGEUSDT', 'DOGE', 'USDT', 'spot', 'active', 7, now()),
  ('binance', 'AVAXUSDT', 'AVAX', 'USDT', 'spot', 'active', 8, now()),
  ('binance', 'LINKUSDT', 'LINK', 'USDT', 'spot', 'active', 9, now()),
  ('binance', 'LTCUSDT', 'LTC', 'USDT', 'spot', 'active', 10, now()),
  ('okx', 'BTC-USDT', 'BTC', 'USDT', 'spot', 'active', 1, now()),
  ('okx', 'ETH-USDT', 'ETH', 'USDT', 'spot', 'active', 2, now()),
  ('okx', 'SOL-USDT', 'SOL', 'USDT', 'spot', 'active', 3, now()),
  ('okx', 'OKB-USDT', 'OKB', 'USDT', 'spot', 'active', 4, now()),
  ('okx', 'XRP-USDT', 'XRP', 'USDT', 'spot', 'active', 5, now()),
  ('okx', 'ADA-USDT', 'ADA', 'USDT', 'spot', 'active', 6, now()),
  ('okx', 'DOGE-USDT', 'DOGE', 'USDT', 'spot', 'active', 7, now()),
  ('okx', 'AVAX-USDT', 'AVAX', 'USDT', 'spot', 'active', 8, now()),
  ('okx', 'LINK-USDT', 'LINK', 'USDT', 'spot', 'active', 9, now()),
  ('okx', 'LTC-USDT', 'LTC', 'USDT', 'spot', 'active', 10, now())
ON CONFLICT (exchange, symbol) DO UPDATE
   SET base_asset = EXCLUDED.base_asset,
       quote_asset = EXCLUDED.quote_asset,
       instrument_type = EXCLUDED.instrument_type,
       status = EXCLUDED.status,
       search_priority = EXCLUDED.search_priority,
       synced_at = EXCLUDED.synced_at,
       updated_at = now();
