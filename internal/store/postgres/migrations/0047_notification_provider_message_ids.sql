ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS provider_message_id text NOT NULL DEFAULT '';

ALTER TABLE notification_outbox
  ADD COLUMN IF NOT EXISTS provider_message_id text NOT NULL DEFAULT '';

ALTER TABLE notifications
  ADD CONSTRAINT notifications_provider_message_id_length_check
    CHECK (char_length(provider_message_id) <= 256) NOT VALID;

ALTER TABLE notification_outbox
  ADD CONSTRAINT notification_outbox_provider_message_id_length_check
    CHECK (char_length(provider_message_id) <= 256) NOT VALID;
