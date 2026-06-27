ALTER TABLE notifications
  DROP CONSTRAINT IF EXISTS notifications_provider_check,
  ADD CONSTRAINT notifications_provider_check
    CHECK (provider IN ('local', 'webhook-demo', 'webhook', 'email', 'telegram', 'feishu'));

ALTER TABLE notification_channels
  DROP CONSTRAINT IF EXISTS notification_channels_provider_check,
  ADD CONSTRAINT notification_channels_provider_check
    CHECK (provider IN ('local', 'webhook-demo', 'webhook', 'email', 'telegram', 'feishu'));

ALTER TABLE notification_outbox
  DROP CONSTRAINT IF EXISTS notification_outbox_provider_check,
  ADD CONSTRAINT notification_outbox_provider_check
    CHECK (provider IN ('local', 'webhook-demo', 'webhook', 'email', 'telegram', 'feishu'));
