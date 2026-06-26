CREATE TABLE IF NOT EXISTS notification_channels (
  id text PRIMARY KEY,
  name text NOT NULL,
  provider text NOT NULL,
  target text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exchange_accounts (
  id text PRIMARY KEY,
  exchange text NOT NULL,
  alias text NOT NULL,
  encrypted_api_key text NOT NULL,
  encrypted_api_secret text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_exchange_accounts_lookup
  ON exchange_accounts (exchange, alias);

CREATE TABLE IF NOT EXISTS operators (
  id text PRIMARY KEY,
  username text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
