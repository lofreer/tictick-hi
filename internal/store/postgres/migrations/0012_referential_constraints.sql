ALTER TABLE trading_tasks
  ADD CONSTRAINT trading_tasks_id_type_unique
    UNIQUE (id, type);

ALTER TABLE strategy_intents
  ADD CONSTRAINT strategy_intents_id_task_unique
    UNIQUE (id, task_id);

ALTER TABLE orders
  ADD CONSTRAINT orders_id_task_unique
    UNIQUE (id, task_id),
  ADD CONSTRAINT orders_trading_task_fk
    FOREIGN KEY (task_id, task_type)
    REFERENCES trading_tasks (id, type)
    ON DELETE CASCADE,
  ADD CONSTRAINT orders_intent_task_fk
    FOREIGN KEY (intent_id, task_id)
    REFERENCES strategy_intents (id, task_id);

ALTER TABLE executions
  ADD CONSTRAINT executions_trading_task_fk
    FOREIGN KEY (task_id, task_type)
    REFERENCES trading_tasks (id, type)
    ON DELETE CASCADE,
  ADD CONSTRAINT executions_order_task_fk
    FOREIGN KEY (order_id, task_id)
    REFERENCES orders (id, task_id)
    ON DELETE CASCADE,
  ADD CONSTRAINT executions_intent_task_fk
    FOREIGN KEY (intent_id, task_id)
    REFERENCES strategy_intents (id, task_id);

ALTER TABLE positions
  ADD CONSTRAINT positions_trading_task_fk
    FOREIGN KEY (task_id, task_type)
    REFERENCES trading_tasks (id, type)
    ON DELETE CASCADE;

ALTER TABLE notifications
  ADD CONSTRAINT notifications_id_task_unique
    UNIQUE (id, task_id),
  ADD CONSTRAINT notifications_trading_task_fk
    FOREIGN KEY (task_id)
    REFERENCES trading_tasks (id)
    ON DELETE CASCADE,
  ADD CONSTRAINT notifications_intent_task_fk
    FOREIGN KEY (intent_id, task_id)
    REFERENCES strategy_intents (id, task_id);

ALTER TABLE notification_outbox
  ADD CONSTRAINT notification_outbox_trading_task_fk
    FOREIGN KEY (task_id)
    REFERENCES trading_tasks (id)
    ON DELETE CASCADE,
  ADD CONSTRAINT notification_outbox_notification_task_fk
    FOREIGN KEY (notification_id, task_id)
    REFERENCES notifications (id, task_id)
    ON DELETE CASCADE,
  ADD CONSTRAINT notification_outbox_intent_task_fk
    FOREIGN KEY (intent_id, task_id)
    REFERENCES strategy_intents (id, task_id);

ALTER TABLE backtest_orders
  ADD CONSTRAINT backtest_orders_intent_task_fk
    FOREIGN KEY (intent_id, backtest_id)
    REFERENCES strategy_intents (id, task_id);
