ALTER TABLE operators
  ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT 'admin';

UPDATE operators
   SET role = 'admin'
 WHERE role IS NULL OR btrim(role) = '';

DO $$
BEGIN
  ALTER TABLE operators
    ADD CONSTRAINT operators_role_check
      CHECK (role IN ('admin', 'operator')) NOT VALID;
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_operators_role_enabled
  ON operators (role, enabled)
  WHERE enabled = true;
