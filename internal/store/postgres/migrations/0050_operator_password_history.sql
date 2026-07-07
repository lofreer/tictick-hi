CREATE TABLE IF NOT EXISTS operator_password_history (
  id text PRIMARY KEY,
  operator_id text NOT NULL REFERENCES operators(id) ON DELETE CASCADE,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT operator_password_history_password_hash_trimmed_check
    CHECK (btrim(password_hash) <> '')
);

CREATE INDEX IF NOT EXISTS idx_operator_password_history_operator_created_at
  ON operator_password_history (operator_id, created_at DESC, id DESC);
