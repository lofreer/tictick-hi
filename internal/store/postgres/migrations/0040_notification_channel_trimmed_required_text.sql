ALTER TABLE notification_channels
  ADD CONSTRAINT notification_channels_trimmed_required_text_check
    CHECK (btrim(name) <> '' AND btrim(target) <> '') NOT VALID;
