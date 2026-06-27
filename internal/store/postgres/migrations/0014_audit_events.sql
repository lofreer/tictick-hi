CREATE TABLE IF NOT EXISTS audit_events (
  id text PRIMARY KEY,
  actor_operator_id text REFERENCES operators(id) ON DELETE SET NULL,
  actor_username text NOT NULL DEFAULT '',
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id text NOT NULL DEFAULT '',
  outcome text NOT NULL,
  request_method text NOT NULL DEFAULT '',
  request_path text NOT NULL DEFAULT '',
  remote_addr text NOT NULL DEFAULT '',
  user_agent text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT audit_events_required_text_check
    CHECK (id <> '' AND action <> '' AND resource_type <> '' AND outcome <> ''),
  CONSTRAINT audit_events_outcome_check
    CHECK (outcome IN ('success', 'failure'))
);

CREATE INDEX IF NOT EXISTS idx_audit_events_created_at
  ON audit_events (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_events_actor_created_at
  ON audit_events (actor_operator_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_events_action_created_at
  ON audit_events (action, created_at DESC);
