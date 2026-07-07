package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

type auditEventHashPrunePlan struct {
	Pruned   []auditEventHashRecord
	Anchor   auditEventHashRecord
	Retained auditEventHashRecord
}

func (store *Store) PruneAuditEvents(
	ctx context.Context,
	request data.AuditEventRetentionRequest,
) (data.AuditEventRetentionResult, error) {
	cutoff := request.Before.UTC()
	result := data.AuditEventRetentionResult{
		DryRun:  request.DryRun,
		Cutoff:  cutoff,
		Message: "no audit events eligible for pruning",
	}
	if cutoff.IsZero() {
		return result, fmt.Errorf("audit retention cutoff is required")
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin audit retention transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, auditEventHashChainLockKey); err != nil {
		return result, fmt.Errorf("lock audit event hash chain: %w", err)
	}
	verification, err := verifyAuditEventHashChain(ctx, tx, time.Now().UTC())
	if err != nil {
		return result, err
	}
	if verification.Status == "failure" {
		return result, fmt.Errorf("audit event hash chain verification failed: %s", verification.Message)
	}

	records, err := listAuditEventHashRecords(ctx, tx)
	if err != nil {
		return result, err
	}
	anchors, err := listAuditEventRetentionAnchorHashes(ctx, tx)
	if err != nil {
		return result, err
	}
	plan, err := planAuditEventHashPrune(records, anchors, cutoff)
	if err != nil {
		return result, err
	}
	legacyCount, err := countLegacyAuditEventsBefore(ctx, tx, cutoff)
	if err != nil {
		return result, err
	}

	result.LegacyPrunedCount = legacyCount
	result.HashedPrunedCount = len(plan.Pruned)
	result.PrunedCount = result.LegacyPrunedCount + result.HashedPrunedCount
	if result.HashedPrunedCount > 0 {
		result.AnchorHash = plan.Anchor.EventHash
		result.AnchorEventID = plan.Anchor.ID
		result.RetainedEventID = plan.Retained.ID
	}
	result.Message = auditRetentionResultMessage(result)
	if request.DryRun {
		return result, nil
	}
	if result.PrunedCount == 0 {
		if err := tx.Commit(ctx); err != nil {
			return result, fmt.Errorf("commit audit retention transaction: %w", err)
		}
		return result, nil
	}

	if result.HashedPrunedCount > 0 {
		anchorID, err := core.NewPrefixedID("aea")
		if err != nil {
			return result, err
		}
		result.AnchorID = anchorID
		if err := insertAuditEventRetentionAnchor(ctx, tx, result); err != nil {
			return result, err
		}
		if err := deleteHashedAuditEvents(ctx, tx, plan.Pruned); err != nil {
			return result, err
		}
	}
	if result.LegacyPrunedCount > 0 {
		if err := deleteLegacyAuditEventsBefore(ctx, tx, cutoff, result.LegacyPrunedCount); err != nil {
			return result, err
		}
	}

	verification, err = verifyAuditEventHashChain(ctx, tx, time.Now().UTC())
	if err != nil {
		return result, err
	}
	if verification.Status == "failure" {
		return result, fmt.Errorf("audit event hash chain verification failed after prune: %s", verification.Message)
	}
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit audit retention transaction: %w", err)
	}
	result.Message = auditRetentionResultMessage(result)
	return result, nil
}

func countLegacyAuditEventsBefore(ctx context.Context, db auditEventHashQueryer, cutoff time.Time) (int, error) {
	var count int
	if err := db.QueryRow(ctx, `
			SELECT count(*)
			  FROM audit_events
			 WHERE event_hash = ''
			   AND created_at < $1`, cutoff).Scan(&count); err != nil {
		return 0, fmt.Errorf("count legacy audit events before retention cutoff: %w", err)
	}
	return count, nil
}

func planAuditEventHashPrune(
	records []auditEventHashRecord,
	anchors map[string]struct{},
	cutoff time.Time,
) (auditEventHashPrunePlan, error) {
	ordered, err := orderAuditEventHashRecords(records, anchors)
	if err != nil {
		return auditEventHashPrunePlan{}, err
	}
	plan := auditEventHashPrunePlan{}
	for _, record := range ordered {
		if !record.CreatedAt.Before(cutoff) {
			break
		}
		plan.Pruned = append(plan.Pruned, record)
	}
	if len(plan.Pruned) == 0 {
		return plan, nil
	}
	if len(plan.Pruned) == len(ordered) {
		return auditEventHashPrunePlan{}, fmt.Errorf("audit retention cutoff would prune every hashed audit event")
	}
	plan.Anchor = plan.Pruned[len(plan.Pruned)-1]
	plan.Retained = ordered[len(plan.Pruned)]
	return plan, nil
}

func orderAuditEventHashRecords(
	records []auditEventHashRecord,
	anchors map[string]struct{},
) ([]auditEventHashRecord, error) {
	if len(records) == 0 {
		return nil, nil
	}
	byHash := make(map[string]auditEventHashRecord, len(records))
	childByPrevious := make(map[string]auditEventHashRecord, len(records))
	for _, record := range records {
		byHash[record.EventHash] = record
		if record.PreviousHash == "" {
			continue
		}
		if _, exists := childByPrevious[record.PreviousHash]; exists {
			return nil, fmt.Errorf("audit event hash chain forks")
		}
		childByPrevious[record.PreviousHash] = record
	}

	var root auditEventHashRecord
	rootCount := 0
	for _, record := range records {
		if record.PreviousHash == "" {
			root = record
			rootCount++
			continue
		}
		if _, exists := byHash[record.PreviousHash]; exists {
			continue
		}
		if _, anchored := anchors[record.PreviousHash]; anchored {
			root = record
			rootCount++
			continue
		}
		return nil, fmt.Errorf("audit event previous hash is missing")
	}
	if rootCount != 1 {
		return nil, fmt.Errorf("audit event hash chain must have exactly one root")
	}

	ordered := make([]auditEventHashRecord, 0, len(records))
	visited := map[string]bool{}
	for current := root; current.EventHash != ""; {
		if visited[current.EventHash] {
			return nil, fmt.Errorf("audit event hash chain contains a cycle")
		}
		visited[current.EventHash] = true
		ordered = append(ordered, current)
		child, exists := childByPrevious[current.EventHash]
		if !exists {
			break
		}
		current = child
	}
	if len(ordered) != len(records) {
		return nil, fmt.Errorf("audit event hash chain is disconnected")
	}
	return ordered, nil
}

func insertAuditEventRetentionAnchor(
	ctx context.Context,
	tx auditEventExecer,
	result data.AuditEventRetentionResult,
) error {
	_, err := tx.Exec(ctx, `
			INSERT INTO audit_event_retention_anchors (
				id, anchor_hash, anchor_event_id, retained_event_id, retention_cutoff, pruned_count
			)
			VALUES ($1, $2, $3, $4, $5, $6)`,
		result.AnchorID,
		result.AnchorHash,
		result.AnchorEventID,
		result.RetainedEventID,
		result.Cutoff,
		result.HashedPrunedCount,
	)
	if err != nil {
		return fmt.Errorf("record audit event retention anchor: %w", err)
	}
	return nil
}

func deleteHashedAuditEvents(ctx context.Context, tx auditEventExecer, records []auditEventHashRecord) error {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM audit_events WHERE id = ANY($1::text[])`, ids)
	if err != nil {
		return fmt.Errorf("delete hashed audit events: %w", err)
	}
	if int(tag.RowsAffected()) != len(ids) {
		return fmt.Errorf("delete hashed audit events affected %d rows, want %d", tag.RowsAffected(), len(ids))
	}
	return nil
}

func deleteLegacyAuditEventsBefore(ctx context.Context, tx auditEventExecer, cutoff time.Time, expected int) error {
	tag, err := tx.Exec(ctx, `
			DELETE FROM audit_events
			 WHERE event_hash = ''
			   AND created_at < $1`, cutoff)
	if err != nil {
		return fmt.Errorf("delete legacy audit events before retention cutoff: %w", err)
	}
	if int(tag.RowsAffected()) != expected {
		return fmt.Errorf("delete legacy audit events affected %d rows, want %d", tag.RowsAffected(), expected)
	}
	return nil
}

func auditRetentionResultMessage(result data.AuditEventRetentionResult) string {
	if result.PrunedCount == 0 {
		return "no audit events eligible for pruning"
	}
	action := "pruned"
	if result.DryRun {
		action = "would prune"
	}
	return fmt.Sprintf(
		"%s %d audit events: %d hashed and %d legacy",
		action,
		result.PrunedCount,
		result.HashedPrunedCount,
		result.LegacyPrunedCount,
	)
}
