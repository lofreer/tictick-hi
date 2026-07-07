ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS traceparent text;

ALTER TABLE notification_outbox
  ADD COLUMN IF NOT EXISTS traceparent text;

CREATE INDEX IF NOT EXISTS idx_notifications_traceparent
  ON notifications (traceparent)
  WHERE traceparent IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_notification_outbox_traceparent
  ON notification_outbox (traceparent)
  WHERE traceparent IS NOT NULL;
