package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

type auditEventHashRecord struct {
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
	PreviousHash    string
	EventHash       string
	CreatedAt       time.Time
}

func (store *Store) VerifyAuditEventHashChain(ctx context.Context) (data.AuditEventHashChainVerification, error) {
	var skippedCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE event_hash = ''`).Scan(&skippedCount); err != nil {
		return data.AuditEventHashChainVerification{}, fmt.Errorf("count unhashed audit events: %w", err)
	}
	anchorHashes, err := store.listAuditEventRetentionAnchorHashes(ctx)
	if err != nil {
		return data.AuditEventHashChainVerification{}, err
	}
	rows, err := store.pool.Query(ctx, `
			SELECT id, coalesce(actor_operator_id, ''), actor_username, action, resource_type,
			       resource_id, outcome, request_method, request_path, remote_addr,
			       user_agent, metadata, previous_hash, event_hash, created_at
			  FROM audit_events
			 WHERE event_hash <> ''
		 ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return data.AuditEventHashChainVerification{}, fmt.Errorf("list hashed audit events: %w", err)
	}
	defer rows.Close()

	records, err := pgx.CollectRows(rows, scanAuditEventHashRecord)
	if err != nil {
		return data.AuditEventHashChainVerification{}, fmt.Errorf("collect hashed audit events: %w", err)
	}
	return verifyAuditEventHashRecords(records, skippedCount, anchorHashes, time.Now().UTC())
}

func (store *Store) listAuditEventRetentionAnchorHashes(ctx context.Context) (map[string]struct{}, error) {
	rows, err := store.pool.Query(ctx, `SELECT anchor_hash FROM audit_event_retention_anchors`)
	if err != nil {
		return nil, fmt.Errorf("list audit event retention anchors: %w", err)
	}
	defer rows.Close()

	anchors := map[string]struct{}{}
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, fmt.Errorf("scan audit event retention anchor: %w", err)
		}
		anchors[hash] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collect audit event retention anchors: %w", err)
	}
	return anchors, nil
}

func scanAuditEventHashRecord(row pgx.CollectableRow) (auditEventHashRecord, error) {
	var record auditEventHashRecord
	err := row.Scan(
		&record.ID,
		&record.ActorOperatorID,
		&record.ActorUsername,
		&record.Action,
		&record.ResourceType,
		&record.ResourceID,
		&record.Outcome,
		&record.RequestMethod,
		&record.RequestPath,
		&record.RemoteAddr,
		&record.UserAgent,
		&record.Metadata,
		&record.PreviousHash,
		&record.EventHash,
		&record.CreatedAt,
	)
	return record, err
}

func verifyAuditEventHashRecords(
	records []auditEventHashRecord,
	skippedCount int,
	retentionAnchorHashes map[string]struct{},
	checkedAt time.Time,
) (data.AuditEventHashChainVerification, error) {
	result := data.AuditEventHashChainVerification{
		Status:       "ok",
		CheckedCount: len(records),
		SkippedCount: skippedCount,
		CheckedAt:    checkedAt.UTC(),
		Message:      "audit event hash chain is valid",
	}
	if skippedCount > 0 {
		result.Status = "warning"
		result.Message = "audit event hash chain is valid for hashed events; legacy unhashed events were skipped"
	}
	if len(records) == 0 {
		if skippedCount > 0 {
			result.Message = "no hashed audit events found; legacy unhashed events were skipped"
		}
		return result, nil
	}

	byHash := make(map[string]auditEventHashRecord, len(records))
	for _, record := range records {
		expected, err := recomputeAuditEventHash(record)
		if err != nil {
			return failedAuditEventHashVerification(result, record.ID, err.Error()), nil
		}
		if record.EventHash != expected {
			return failedAuditEventHashVerification(result, record.ID, "audit event hash mismatch"), nil
		}
		if _, exists := byHash[record.EventHash]; exists {
			return failedAuditEventHashVerification(result, record.ID, "duplicate audit event hash"), nil
		}
		byHash[record.EventHash] = record
	}

	var root auditEventHashRecord
	childByPrevious := make(map[string]auditEventHashRecord, len(records))
	for _, record := range records {
		if record.PreviousHash == "" {
			result.RootCount++
			root = record
			continue
		}
		if _, exists := childByPrevious[record.PreviousHash]; exists {
			return failedAuditEventHashVerification(result, record.ID, "audit event hash chain forks"), nil
		}
		childByPrevious[record.PreviousHash] = record
		if _, exists := byHash[record.PreviousHash]; !exists {
			if _, anchored := retentionAnchorHashes[record.PreviousHash]; !anchored {
				return failedAuditEventHashVerification(result, record.ID, "audit event previous hash is missing"), nil
			}
			result.RootCount++
			root = record
		}
	}
	if result.RootCount != 1 {
		return failedAuditEventHashVerification(result, "", "audit event hash chain must have exactly one root"), nil
	}

	visited := map[string]bool{}
	for current := root; current.EventHash != ""; {
		if visited[current.EventHash] {
			return failedAuditEventHashVerification(result, current.ID, "audit event hash chain contains a cycle"), nil
		}
		visited[current.EventHash] = true
		child, exists := childByPrevious[current.EventHash]
		if !exists {
			result.TailCount++
			break
		}
		current = child
	}
	if len(visited) != len(records) {
		return failedAuditEventHashVerification(result, "", "audit event hash chain is disconnected"), nil
	}
	if result.TailCount != 1 {
		return failedAuditEventHashVerification(result, "", "audit event hash chain must have exactly one tail"), nil
	}
	return result, nil
}

func recomputeAuditEventHash(record auditEventHashRecord) (string, error) {
	metadata, err := canonicalAuditEventMetadata(record.Metadata)
	if err != nil {
		return "", err
	}
	return auditEventHash(auditEventHashInput{
		PreviousHash:    record.PreviousHash,
		ID:              record.ID,
		ActorOperatorID: record.ActorOperatorID,
		ActorUsername:   record.ActorUsername,
		Action:          record.Action,
		ResourceType:    record.ResourceType,
		ResourceID:      record.ResourceID,
		Outcome:         record.Outcome,
		RequestMethod:   record.RequestMethod,
		RequestPath:     record.RequestPath,
		RemoteAddr:      record.RemoteAddr,
		UserAgent:       record.UserAgent,
		Metadata:        metadata,
		CreatedAt:       record.CreatedAt,
	})
}

func canonicalAuditEventMetadata(metadata []byte) ([]byte, error) {
	if len(metadata) == 0 {
		return []byte(`{}`), nil
	}
	values := map[string]string{}
	if err := json.Unmarshal(metadata, &values); err != nil {
		return nil, fmt.Errorf("decode audit metadata for hash verification: %w", err)
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("encode audit metadata for hash verification: %w", err)
	}
	return encoded, nil
}

func failedAuditEventHashVerification(
	result data.AuditEventHashChainVerification,
	brokenEventID string,
	message string,
) data.AuditEventHashChainVerification {
	result.Status = "failure"
	result.BrokenEventID = brokenEventID
	result.Message = message
	return result
}
