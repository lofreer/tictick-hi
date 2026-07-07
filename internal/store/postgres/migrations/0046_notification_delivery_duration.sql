ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS last_delivery_duration_ms bigint;

ALTER TABLE notification_outbox
  ADD COLUMN IF NOT EXISTS last_delivery_duration_ms bigint;

ALTER TABLE notifications
  ADD CONSTRAINT notifications_delivery_duration_non_negative_check
    CHECK (last_delivery_duration_ms IS NULL OR last_delivery_duration_ms >= 0) NOT VALID;

ALTER TABLE notification_outbox
  ADD CONSTRAINT notification_outbox_delivery_duration_non_negative_check
    CHECK (last_delivery_duration_ms IS NULL OR last_delivery_duration_ms >= 0) NOT VALID;
