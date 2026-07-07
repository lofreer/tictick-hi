ALTER TABLE operator_sessions
  ADD COLUMN IF NOT EXISTS remote_addr text,
  ADD COLUMN IF NOT EXISTS user_agent text;
