CREATE OR REPLACE FUNCTION enforce_data_sync_task_status_transition()
RETURNS trigger AS $$
BEGIN
  IF NEW.status = OLD.status THEN
    RETURN NEW;
  END IF;

  IF NOT (
    (OLD.status = 'pending' AND NEW.status IN ('running', 'paused'))
    OR (OLD.status = 'running' AND NEW.status IN ('pending', 'paused', 'succeeded', 'failed'))
    OR (OLD.status = 'paused' AND NEW.status IN ('pending', 'running'))
    OR (OLD.status = 'failed' AND NEW.status = 'pending')
  ) THEN
    RAISE EXCEPTION 'data_sync_tasks_status_transition_check: % -> %', OLD.status, NEW.status
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS data_sync_tasks_status_transition_guard ON data_sync_tasks;
CREATE TRIGGER data_sync_tasks_status_transition_guard
  BEFORE UPDATE OF status ON data_sync_tasks
  FOR EACH ROW
  EXECUTE FUNCTION enforce_data_sync_task_status_transition();

CREATE OR REPLACE FUNCTION enforce_backtest_task_status_transition()
RETURNS trigger AS $$
BEGIN
  IF NEW.status = OLD.status THEN
    RETURN NEW;
  END IF;

  IF NOT (
    (OLD.status = 'pending' AND NEW.status = 'running')
    OR (OLD.status = 'running' AND NEW.status IN ('pending', 'succeeded', 'failed'))
  ) THEN
    RAISE EXCEPTION 'backtest_tasks_status_transition_check: % -> %', OLD.status, NEW.status
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS backtest_tasks_status_transition_guard ON backtest_tasks;
CREATE TRIGGER backtest_tasks_status_transition_guard
  BEFORE UPDATE OF status ON backtest_tasks
  FOR EACH ROW
  EXECUTE FUNCTION enforce_backtest_task_status_transition();

CREATE OR REPLACE FUNCTION enforce_trading_task_status_transition()
RETURNS trigger AS $$
BEGIN
  IF NEW.status = OLD.status THEN
    RETURN NEW;
  END IF;

  IF NOT (
    (OLD.status = 'pending' AND NEW.status IN ('running', 'paused'))
    OR (OLD.status = 'running' AND NEW.status IN ('paused', 'failed'))
    OR (OLD.status = 'paused' AND NEW.status = 'running')
  ) THEN
    RAISE EXCEPTION 'trading_tasks_status_transition_check: % -> %', OLD.status, NEW.status
      USING ERRCODE = '23514';
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trading_tasks_status_transition_guard ON trading_tasks;
CREATE TRIGGER trading_tasks_status_transition_guard
  BEFORE UPDATE OF status ON trading_tasks
  FOR EACH ROW
  EXECUTE FUNCTION enforce_trading_task_status_transition();
