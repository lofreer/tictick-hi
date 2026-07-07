CREATE TABLE IF NOT EXISTS audit_event_retention_anchors (
  id text PRIMARY KEY,
  anchor_hash text NOT NULL,
  anchor_event_id text NOT NULL,
  retained_event_id text NOT NULL,
  retention_cutoff timestamptz NOT NULL,
  pruned_count integer NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT audit_event_retention_anchors_required_text_check
    CHECK (
      btrim(id) <> '' AND
      btrim(anchor_event_id) <> '' AND
      btrim(retained_event_id) <> ''
    ),
  CONSTRAINT audit_event_retention_anchors_hash_check
    CHECK (anchor_hash ~ '^[0-9a-f]{64}$'),
  CONSTRAINT audit_event_retention_anchors_count_check
    CHECK (pruned_count > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_event_retention_anchors_hash
  ON audit_event_retention_anchors (anchor_hash);
