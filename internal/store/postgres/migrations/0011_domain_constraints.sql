ALTER TABLE data_sync_tasks
  ADD CONSTRAINT data_sync_tasks_status_check
    CHECK (status IN ('pending', 'running', 'stopping', 'paused', 'succeeded', 'failed', 'cancelled')),
  ADD CONSTRAINT data_sync_tasks_attempt_count_check
    CHECK (attempt_count >= 0),
  ADD CONSTRAINT data_sync_tasks_time_range_check
    CHECK (start_time IS NULL OR end_time IS NULL OR start_time <= end_time);

ALTER TABLE market_candles
  ADD CONSTRAINT market_candles_time_range_check
    CHECK (open_time < close_time),
  ADD CONSTRAINT market_candles_non_negative_values_check
    CHECK (open >= 0 AND high >= 0 AND low >= 0 AND close >= 0 AND volume >= 0),
  ADD CONSTRAINT market_candles_ohlc_bounds_check
    CHECK (high >= GREATEST(open, close, low) AND low <= LEAST(open, close, high));

ALTER TABLE backtest_tasks
  ADD CONSTRAINT backtest_tasks_status_check
    CHECK (status IN ('pending', 'running', 'stopping', 'paused', 'succeeded', 'failed', 'cancelled')),
  ADD CONSTRAINT backtest_tasks_trigger_mode_check
    CHECK (trigger_mode IN ('closed_candle', 'minute_replay')),
  ADD CONSTRAINT backtest_tasks_time_range_check
    CHECK (start_time IS NULL OR end_time IS NULL OR start_time <= end_time),
  ADD CONSTRAINT backtest_tasks_decimal_bounds_check
    CHECK (initial_balance > 0 AND fee_bps >= 0 AND slippage_bps >= 0),
  ADD CONSTRAINT backtest_tasks_attempt_count_check
    CHECK (attempt_count >= 0);

ALTER TABLE backtest_orders
  ADD CONSTRAINT backtest_orders_side_check
    CHECK (side IN ('buy', 'sell')),
  ADD CONSTRAINT backtest_orders_status_check
    CHECK (status IN ('filled')),
  ADD CONSTRAINT backtest_orders_decimal_bounds_check
    CHECK (price >= 0 AND quantity > 0);

ALTER TABLE trading_tasks
  ADD CONSTRAINT trading_tasks_type_check
    CHECK (type IN ('paper', 'live')),
  ADD CONSTRAINT trading_tasks_status_check
    CHECK (status IN ('pending', 'running', 'stopping', 'paused', 'succeeded', 'failed', 'cancelled')),
  ADD CONSTRAINT trading_tasks_attempt_count_check
    CHECK (attempt_count >= 0);

ALTER TABLE strategy_intents
  ADD CONSTRAINT strategy_intents_task_type_check
    CHECK (task_type IN ('backtest', 'paper', 'live')),
  ADD CONSTRAINT strategy_intents_intent_type_check
    CHECK (intent_type IN ('order', 'notification')),
  ADD CONSTRAINT strategy_intents_policy_check
    CHECK (policy IN ('simulate', 'execute', 'notify')),
  ADD CONSTRAINT strategy_intents_status_check
    CHECK (status IN ('accepted', 'executed', 'notification_pending', 'failed'));

ALTER TABLE orders
  ADD CONSTRAINT orders_task_type_check
    CHECK (task_type IN ('paper', 'live')),
  ADD CONSTRAINT orders_side_check
    CHECK (side IN ('buy', 'sell')),
  ADD CONSTRAINT orders_order_type_check
    CHECK (order_type IN ('market', 'limit')),
  ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'submitted', 'filled', 'failed', 'cancelled', 'rejected')),
  ADD CONSTRAINT orders_decimal_bounds_check
    CHECK (price >= 0 AND quantity > 0);

ALTER TABLE notifications
  ADD CONSTRAINT notifications_status_check
    CHECK (status IN ('pending', 'running', 'sent', 'failed', 'retry_scheduled')),
  ADD CONSTRAINT notifications_provider_check
    CHECK (provider IN ('local', 'webhook-demo', 'webhook')),
  ADD CONSTRAINT notifications_attempt_bounds_check
    CHECK (attempt_count >= 0 AND max_attempts > 0);

ALTER TABLE notification_channels
  ADD CONSTRAINT notification_channels_provider_check
    CHECK (provider IN ('local', 'webhook-demo', 'webhook')),
  ADD CONSTRAINT notification_channels_required_text_check
    CHECK (name <> '' AND target <> '');

ALTER TABLE exchange_accounts
  ADD CONSTRAINT exchange_accounts_required_text_check
    CHECK (exchange <> '' AND alias <> '' AND encrypted_api_key <> '' AND encrypted_api_secret <> '');

ALTER TABLE operators
  ADD CONSTRAINT operators_required_text_check
    CHECK (username <> '' AND password_hash <> '');

ALTER TABLE notification_outbox
  ADD CONSTRAINT notification_outbox_status_check
    CHECK (status IN ('pending', 'running', 'retry_scheduled', 'delivered', 'failed')),
  ADD CONSTRAINT notification_outbox_provider_check
    CHECK (provider IN ('local', 'webhook-demo', 'webhook')),
  ADD CONSTRAINT notification_outbox_attempt_bounds_check
    CHECK (attempt_count >= 0 AND max_attempts > 0),
  ADD CONSTRAINT notification_outbox_next_attempt_check
    CHECK (status IN ('delivered', 'failed') OR next_attempt_at IS NOT NULL);

ALTER TABLE executions
  ADD CONSTRAINT executions_task_type_check
    CHECK (task_type IN ('paper', 'live')),
  ADD CONSTRAINT executions_side_check
    CHECK (side IN ('buy', 'sell')),
  ADD CONSTRAINT executions_status_check
    CHECK (status IN ('filled')),
  ADD CONSTRAINT executions_decimal_bounds_check
    CHECK (price >= 0 AND quantity > 0 AND fee >= 0);

ALTER TABLE positions
  ADD CONSTRAINT positions_task_type_check
    CHECK (task_type IN ('paper', 'live')),
  ADD CONSTRAINT positions_average_price_check
    CHECK (average_price >= 0);
