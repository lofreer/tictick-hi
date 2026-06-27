ALTER TABLE notification_outbox
  ALTER COLUMN next_attempt_at DROP NOT NULL;

