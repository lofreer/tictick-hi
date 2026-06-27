CREATE OR REPLACE FUNCTION restrict_backtest_task_delete_with_strategy_intents()
RETURNS trigger AS $$
BEGIN
  IF EXISTS (
    SELECT 1
      FROM strategy_intents
     WHERE task_id = OLD.id
       AND task_type = 'backtest'
  ) THEN
    RAISE EXCEPTION 'backtest task % is still referenced by strategy intents', OLD.id
      USING ERRCODE = '23503',
            CONSTRAINT = 'strategy_intents_backtest_parent_delete_restrict';
  END IF;

  RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER strategy_intents_backtest_parent_delete_restrict
BEFORE DELETE ON backtest_tasks
FOR EACH ROW
EXECUTE FUNCTION restrict_backtest_task_delete_with_strategy_intents();

CREATE OR REPLACE FUNCTION restrict_trading_task_delete_with_strategy_intents()
RETURNS trigger AS $$
BEGIN
  IF EXISTS (
    SELECT 1
      FROM strategy_intents
     WHERE task_id = OLD.id
  ) THEN
    RAISE EXCEPTION 'trading task % is still referenced by strategy intents', OLD.id
      USING ERRCODE = '23503',
            CONSTRAINT = 'strategy_intents_trading_parent_delete_restrict';
  END IF;

  RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER strategy_intents_trading_parent_delete_restrict
BEFORE DELETE ON trading_tasks
FOR EACH ROW
EXECUTE FUNCTION restrict_trading_task_delete_with_strategy_intents();
