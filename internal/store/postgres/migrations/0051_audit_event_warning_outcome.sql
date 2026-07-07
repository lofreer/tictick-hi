ALTER TABLE audit_events
  DROP CONSTRAINT IF EXISTS audit_events_outcome_check;

ALTER TABLE audit_events
  ADD CONSTRAINT audit_events_outcome_check
  CHECK (outcome IN ('success', 'failure', 'warning'));
