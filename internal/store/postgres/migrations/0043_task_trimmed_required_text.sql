ALTER TABLE data_sync_tasks
  ADD CONSTRAINT data_sync_tasks_trimmed_required_text_check
    CHECK (
      btrim(exchange) <> '' AND
      btrim(symbol) <> '' AND
      btrim(interval) <> ''
    ) NOT VALID;

ALTER TABLE backtest_tasks
  ADD CONSTRAINT backtest_tasks_trimmed_required_text_check
    CHECK (
      btrim(name) <> '' AND
      btrim(exchange) <> '' AND
      btrim(symbol) <> '' AND
      btrim(interval) <> '' AND
      btrim(strategy_id) <> ''
    ) NOT VALID;

ALTER TABLE trading_tasks
  ADD CONSTRAINT trading_tasks_trimmed_required_text_check
    CHECK (
      btrim(name) <> '' AND
      btrim(exchange) <> '' AND
      btrim(account_id) <> '' AND
      btrim(symbol) <> '' AND
      btrim(interval) <> '' AND
      btrim(strategy_id) <> ''
    ) NOT VALID;
