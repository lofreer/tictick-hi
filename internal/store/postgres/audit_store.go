package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

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

	auditEventHashChainLockKey int64 = 0x41554449545f4843
	auditEventHashVersion            = 1
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
	createdAt := time.Now().UTC().Truncate(time.Microsecond)

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.AuditEvent{}, fmt.Errorf("begin audit event transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, auditEventHashChainLockKey); err != nil {
		return data.AuditEvent{}, fmt.Errorf("lock audit event hash chain: %w", err)
	}

	previousHash, err := auditEventChainTailHash(ctx, tx)
	if err != nil {
		return data.AuditEvent{}, err
	}
	eventHash, err := auditEventHash(auditEventHashInput{
		PreviousHash:    previousHash,
		ID:              id,
		ActorOperatorID: event.ActorOperatorID,
		ActorUsername:   event.ActorUsername,
		Action:          event.Action,
		ResourceType:    event.ResourceType,
		ResourceID:      event.ResourceID,
		Outcome:         event.Outcome,
		RequestMethod:   event.RequestMethod,
		RequestPath:     event.RequestPath,
		RemoteAddr:      event.RemoteAddr,
		UserAgent:       event.UserAgent,
		Metadata:        metadata,
		CreatedAt:       createdAt,
	})
	if err != nil {
		return data.AuditEvent{}, fmt.Errorf("hash audit event: %w", err)
	}

	row := tx.QueryRow(ctx, `
			INSERT INTO audit_events (
				id, actor_operator_id, actor_username, action, resource_type,
				resource_id, outcome, request_method, request_path, remote_addr,
				user_agent, metadata, previous_hash, event_hash, created_at
			)
			VALUES ($1, nullif($2, ''), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13, $14, $15)
			RETURNING id, coalesce(actor_operator_id, ''), actor_username, action, resource_type,
			          resource_id, outcome, request_method, request_path, remote_addr,
			          user_agent, metadata, previous_hash, event_hash, created_at`,
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
		previousHash,
		eventHash,
		createdAt,
	)

	created, err := scanAuditEventRow(row)
	if err != nil {
		return data.AuditEvent{}, fmt.Errorf("record audit event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return data.AuditEvent{}, fmt.Errorf("commit audit event transaction: %w", err)
	}
	return created, nil
}

func (store *Store) ListAuditEvents(ctx context.Context, limit int) ([]data.AuditEvent, error) {
	page, err := store.ListAuditEventPage(ctx, data.AuditEventListQuery{Limit: limit})
	if err != nil {
		return nil, err
	}
	return page.Events, nil
}

func (store *Store) ListAuditEventPage(ctx context.Context, query data.AuditEventListQuery) (data.AuditEventPage, error) {
	limit := normalizeAuditEventLimit(query.Limit)
	dbLimit := limit + 1
	sql := `
			SELECT id, coalesce(actor_operator_id, ''), actor_username, action, resource_type,
			       resource_id, outcome, request_method, request_path, remote_addr,
			       user_agent, metadata, previous_hash, event_hash, created_at
			  FROM audit_events
		 ORDER BY created_at DESC, id DESC
		 LIMIT $1`
	args := []any{dbLimit}
	if query.Cursor != nil {
		sql = `
			SELECT id, coalesce(actor_operator_id, ''), actor_username, action, resource_type,
			       resource_id, outcome, request_method, request_path, remote_addr,
			       user_agent, metadata, previous_hash, event_hash, created_at
			  FROM audit_events
			 WHERE (created_at, id) < ($2, $3)
		 ORDER BY created_at DESC, id DESC
		 LIMIT $1`
		args = append(args, query.Cursor.CreatedAt, query.Cursor.ID)
	}
	rows, err := store.pool.Query(ctx, sql, args...)
	if err != nil {
		return data.AuditEventPage{}, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events, err := pgx.CollectRows(rows, scanAuditEvent)
	if err != nil {
		return data.AuditEventPage{}, fmt.Errorf("collect audit events: %w", err)
	}
	page := data.AuditEventPage{Events: events}
	if len(events) > limit {
		page.Events = events[:limit]
		nextCursor, err := data.EncodeAuditEventCursor(data.NewAuditEventCursor(page.Events[len(page.Events)-1]))
		if err != nil {
			return data.AuditEventPage{}, fmt.Errorf("encode audit event cursor: %w", err)
		}
		page.NextCursor = nextCursor
	}
	return page, nil
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
		&event.PreviousHash,
		&event.EventHash,
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

type auditEventHashInput struct {
	PreviousHash    string
	ID              string
	ActorOperatorID string
	ActorUsername   string
	Action          string
	ResourceType    string
	ResourceID      string
	Outcome         string
	RequestMethod   string
	RequestPath     string
	RemoteAddr      string
	UserAgent       string
	Metadata        []byte
	CreatedAt       time.Time
}

type auditEventHashPayload struct {
	Version         int             `json:"version"`
	PreviousHash    string          `json:"previousHash"`
	ID              string          `json:"id"`
	ActorOperatorID string          `json:"actorOperatorId"`
	ActorUsername   string          `json:"actorUsername"`
	Action          string          `json:"action"`
	ResourceType    string          `json:"resourceType"`
	ResourceID      string          `json:"resourceId"`
	Outcome         string          `json:"outcome"`
	RequestMethod   string          `json:"requestMethod"`
	RequestPath     string          `json:"requestPath"`
	RemoteAddr      string          `json:"remoteAddr"`
	UserAgent       string          `json:"userAgent"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       string          `json:"createdAt"`
}

func auditEventChainTailHash(ctx context.Context, tx pgx.Tx) (string, error) {
	var previousHash string
	err := tx.QueryRow(ctx, `
			SELECT tail.event_hash
			  FROM audit_events tail
			 WHERE tail.event_hash <> ''
			   AND NOT EXISTS (
			     SELECT 1
			       FROM audit_events child
			      WHERE child.previous_hash = tail.event_hash
			   )
		 ORDER BY tail.created_at DESC, tail.id DESC
		 LIMIT 1`).Scan(&previousHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read audit event hash chain tail: %w", err)
	}
	return previousHash, nil
}

func auditEventHash(input auditEventHashInput) (string, error) {
	metadata := json.RawMessage(input.Metadata)
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	payload := auditEventHashPayload{
		Version:         auditEventHashVersion,
		PreviousHash:    input.PreviousHash,
		ID:              input.ID,
		ActorOperatorID: input.ActorOperatorID,
		ActorUsername:   input.ActorUsername,
		Action:          input.Action,
		ResourceType:    input.ResourceType,
		ResourceID:      input.ResourceID,
		Outcome:         input.Outcome,
		RequestMethod:   input.RequestMethod,
		RequestPath:     input.RequestPath,
		RemoteAddr:      input.RemoteAddr,
		UserAgent:       input.UserAgent,
		Metadata:        metadata,
		CreatedAt:       input.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte("tictick-hi:audit-event:v1\n"), encoded...))
	return hex.EncodeToString(sum[:]), nil
}
