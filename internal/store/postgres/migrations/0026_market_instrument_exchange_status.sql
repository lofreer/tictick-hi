ALTER TABLE market_instruments
  ADD COLUMN IF NOT EXISTS exchange_status text NOT NULL DEFAULT 'unknown';

UPDATE market_instruments
   SET exchange_status = status
 WHERE exchange_status = '' OR exchange_status = 'unknown';

ALTER TABLE market_instruments
  DROP CONSTRAINT IF EXISTS market_instruments_exchange_status_check,
  ADD CONSTRAINT market_instruments_exchange_status_check
    CHECK (exchange_status <> '');
