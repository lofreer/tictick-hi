CREATE OR REPLACE FUNCTION enforce_strategy_intent_task_parent()
RETURNS trigger AS $$
BEGIN
  IF NEW.task_type = 'backtest' THEN
    IF NOT EXISTS (
      SELECT 1
        FROM backtest_tasks
       WHERE id = NEW.task_id
    ) THEN
      RAISE EXCEPTION 'strategy intent % references missing backtest task %', NEW.id, NEW.task_id
        USING ERRCODE = '23503',
              CONSTRAINT = 'strategy_intents_task_parent_fk';
    END IF;
  ELSIF NEW.task_type IN ('paper', 'live') THEN
    IF NOT EXISTS (
      SELECT 1
        FROM trading_tasks
       WHERE id = NEW.task_id
         AND type = NEW.task_type
    ) THEN
      RAISE EXCEPTION 'strategy intent % references missing % trading task %', NEW.id, NEW.task_type, NEW.task_id
        USING ERRCODE = '23503',
              CONSTRAINT = 'strategy_intents_task_parent_fk';
    END IF;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER strategy_intents_task_parent_fk
AFTER INSERT OR UPDATE OF task_id, task_type ON strategy_intents
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW
EXECUTE FUNCTION enforce_strategy_intent_task_parent();
