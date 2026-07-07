ALTER TABLE audit_events
  ADD CONSTRAINT audit_events_trimmed_required_text_check
    CHECK (
      btrim(id) <> '' AND
      btrim(action) <> '' AND
      btrim(resource_type) <> '' AND
      btrim(outcome) <> ''
    ) NOT VALID;
