ALTER TABLE market_candles
  ADD CONSTRAINT market_candles_positive_price_values_check
  CHECK (open > 0 AND high > 0 AND low > 0 AND close > 0 AND volume >= 0)
  NOT VALID;
