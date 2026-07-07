ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS request_id text;

ALTER TABLE notification_outbox
  ADD COLUMN IF NOT EXISTS request_id text;

CREATE INDEX IF NOT EXISTS idx_notifications_request_id
  ON notifications (request_id)
  WHERE request_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_notification_outbox_request_id
  ON notification_outbox (request_id)
  WHERE request_id IS NOT NULL;
