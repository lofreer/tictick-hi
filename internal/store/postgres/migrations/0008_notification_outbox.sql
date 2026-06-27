ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS provider text NOT NULL DEFAULT 'local',
  ADD COLUMN IF NOT EXISTS target text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS attempt_count integer NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_attempts integer NOT NULL DEFAULT 3,
  ADD COLUMN IF NOT EXISTS next_attempt_at timestamptz,
  ADD COLUMN IF NOT EXISTS last_attempt_at timestamptz,
  ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

CREATE TABLE IF NOT EXISTS notification_outbox (
  id text PRIMARY KEY,
  notification_id text NOT NULL UNIQUE REFERENCES notifications(id) ON DELETE CASCADE,
  task_id text NOT NULL,
  intent_id text,
  channel text NOT NULL,
  provider text NOT NULL,
  target text NOT NULL,
  title text NOT NULL,
  body text NOT NULL,
  status text NOT NULL DEFAULT 'pending',
  attempt_count integer NOT NULL DEFAULT 0,
  max_attempts integer NOT NULL DEFAULT 3,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  last_attempt_at timestamptz,
  delivered_at timestamptz,
  last_error text,
  locked_by text,
  locked_until timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notification_outbox_claim
  ON notification_outbox (status, next_attempt_at, locked_until);

CREATE INDEX IF NOT EXISTS idx_notification_outbox_task
  ON notification_outbox (task_id, created_at DESC);

INSERT INTO notification_outbox (
  id, notification_id, task_id, intent_id, channel, provider, target,
  title, body, status, attempt_count, max_attempts, next_attempt_at,
  delivered_at, last_error, created_at, updated_at
)
SELECT
  'no_' || n.id,
  n.id,
  n.task_id,
  n.intent_id,
  n.channel,
  n.provider,
  n.target,
  n.title,
  n.body,
  CASE WHEN n.status = 'sent' THEN 'delivered' ELSE n.status END,
  n.attempt_count,
  n.max_attempts,
  COALESCE(n.next_attempt_at, n.created_at),
  n.sent_at,
  n.error,
  n.created_at,
  n.updated_at
FROM notifications n
ON CONFLICT (notification_id) DO NOTHING;
