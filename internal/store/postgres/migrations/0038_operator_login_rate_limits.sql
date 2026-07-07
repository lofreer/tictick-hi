CREATE TABLE IF NOT EXISTS operator_login_rate_limits (
  key_hash text PRIMARY KEY,
  failure_count integer NOT NULL,
  first_failure_at timestamptz NOT NULL,
  locked_until timestamptz,
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT operator_login_rate_limits_key_hash_check
    CHECK (key_hash ~ '^[0-9a-f]{64}$'),
  CONSTRAINT operator_login_rate_limits_failure_count_check
    CHECK (failure_count > 0),
  CONSTRAINT operator_login_rate_limits_locked_until_check
    CHECK (locked_until IS NULL OR locked_until >= first_failure_at),
  CONSTRAINT operator_login_rate_limits_updated_at_check
    CHECK (updated_at >= first_failure_at)
);

CREATE INDEX IF NOT EXISTS idx_operator_login_rate_limits_locked_until
  ON operator_login_rate_limits (locked_until)
  WHERE locked_until IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_operator_login_rate_limits_updated_at
  ON operator_login_rate_limits (updated_at);
