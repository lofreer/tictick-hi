CREATE TABLE IF NOT EXISTS operator_sessions (
  token_hash text PRIMARY KEY,
  operator_id text NOT NULL REFERENCES operators(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_operator_sessions_operator
  ON operator_sessions (operator_id);

CREATE INDEX IF NOT EXISTS idx_operator_sessions_expires_at
  ON operator_sessions (expires_at);
