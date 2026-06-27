ALTER TABLE operator_sessions
  ADD COLUMN IF NOT EXISTS id text;

UPDATE operator_sessions
   SET id = 'os_' || substr(token_hash, 1, 24)
 WHERE id IS NULL OR id = '';

ALTER TABLE operator_sessions
  ALTER COLUMN id SET NOT NULL;

ALTER TABLE operator_sessions
  ADD CONSTRAINT operator_sessions_id_unique
    UNIQUE (id);

ALTER TABLE operator_sessions
  ADD CONSTRAINT operator_sessions_id_required_check
    CHECK (id <> '');
