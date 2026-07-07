package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	defaultAuditEventLimit      = 100
	maxAuditEventLimit          = 500
	maxAuditMetadataEntries     = 20
	maxAuditMetadataKeyLength   = 64
	maxAuditMetadataValueLength = 255
)

func (store *Store) RecordAuditEvent(ctx context.Context, event data.CreateAuditEvent) (data.AuditEvent, error) {
	id, err := core.NewPrefixedID("ae")
	if err != nil {
		return data.AuditEvent{}, err
	}
	metadata, err := json.Marshal(normalizeAuditMetadata(event.Metadata))
	if err != nil {
		return data.AuditEvent{}, fmt.Errorf("marshal audit metadata: %w", err)
	}

	row := store.pool.QueryRow(ctx, `
		INSERT INTO audit_events (
			id, actor_operator_id, actor_username, action, resource_type,
			resource_id, outcome, request_method, request_path, remote_addr,
			user_agent, metadata
		)
		VALUES ($1, nullif($2, ''), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb)
		RETURNING id, coalesce(actor_operator_id, ''), actor_username, action, resource_type,
		          resource_id, outcome, request_method, request_path, remote_addr,
		          user_agent, metadata, created_at`,
		id,
		event.ActorOperatorID,
		event.ActorUsername,
		event.Action,
		event.ResourceType,
		event.ResourceID,
		event.Outcome,
		event.RequestMethod,
		event.RequestPath,
		event.RemoteAddr,
		event.UserAgent,
		string(metadata),
	)

	created, err := scanAuditEventRow(row)
	if err != nil {
		return data.AuditEvent{}, fmt.Errorf("record audit event: %w", err)
	}
	return created, nil
}

func (store *Store) ListAuditEvents(ctx context.Context, limit int) ([]data.AuditEvent, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, coalesce(actor_operator_id, ''), actor_username, action, resource_type,
		       resource_id, outcome, request_method, request_path, remote_addr,
		       user_agent, metadata, created_at
		  FROM audit_events
		 ORDER BY created_at DESC
		 LIMIT $1`,
		normalizeAuditEventLimit(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events, err := pgx.CollectRows(rows, scanAuditEvent)
	if err != nil {
		return nil, fmt.Errorf("collect audit events: %w", err)
	}
	return events, nil
}

func scanAuditEvent(row pgx.CollectableRow) (data.AuditEvent, error) {
	return scanAuditEventRow(row)
}

func scanAuditEventRow(row rowScanner) (data.AuditEvent, error) {
	var event data.AuditEvent
	var metadata []byte
	err := row.Scan(
		&event.ID,
		&event.ActorOperatorID,
		&event.ActorUsername,
		&event.Action,
		&event.ResourceType,
		&event.ResourceID,
		&event.Outcome,
		&event.RequestMethod,
		&event.RequestPath,
		&event.RemoteAddr,
		&event.UserAgent,
		&metadata,
		&event.CreatedAt,
	)
	if err != nil {
		return data.AuditEvent{}, err
	}
	event.Metadata = map[string]string{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
			return data.AuditEvent{}, fmt.Errorf("unmarshal audit metadata: %w", err)
		}
	}
	return event, nil
}

func normalizeAuditEventLimit(limit int) int {
	if limit <= 0 {
		return defaultAuditEventLimit
	}
	if limit > maxAuditEventLimit {
		return maxAuditEventLimit
	}
	return limit
}

func normalizeAuditMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	normalized := make(map[string]string, min(len(metadata), maxAuditMetadataEntries))
	for _, key := range keys {
		normalizedKey := boundedAuditMetadataText(key, maxAuditMetadataKeyLength)
		if normalizedKey == "" {
			continue
		}
		if _, exists := normalized[normalizedKey]; !exists && len(normalized) >= maxAuditMetadataEntries {
			continue
		}
		normalized[normalizedKey] = boundedAuditMetadataText(metadata[key], maxAuditMetadataValueLength)
	}
	return normalized
}

func boundedAuditMetadataText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}
