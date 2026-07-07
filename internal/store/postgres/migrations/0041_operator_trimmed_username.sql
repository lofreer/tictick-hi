ALTER TABLE operators
  ADD CONSTRAINT operators_trimmed_username_check
    CHECK (btrim(username) <> '') NOT VALID;
