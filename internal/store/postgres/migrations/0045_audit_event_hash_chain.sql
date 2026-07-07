ALTER TABLE audit_events
  ADD COLUMN IF NOT EXISTS previous_hash text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS event_hash text NOT NULL DEFAULT '';

ALTER TABLE audit_events
  ADD CONSTRAINT audit_events_hash_format_check
    CHECK (
      (previous_hash = '' OR previous_hash ~ '^[0-9a-f]{64}$') AND
      event_hash ~ '^[0-9a-f]{64}$'
    ) NOT VALID;

CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_events_event_hash_unique
  ON audit_events (event_hash)
  WHERE event_hash <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_events_previous_hash_unique
  ON audit_events (previous_hash)
  WHERE previous_hash <> '';
