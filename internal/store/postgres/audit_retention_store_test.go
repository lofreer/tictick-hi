package postgres

import (
	"strings"
	"testing"
	"time"
)

func TestPlanAuditEventHashPruneAnchorsSafePrefix(t *testing.T) {
	first := testAuditHashRecordAt(t, "ae_1", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	second := testAuditHashRecordAt(t, "ae_2", first.EventHash, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	third := testAuditHashRecordAt(t, "ae_3", second.EventHash, time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC))

	plan, err := planAuditEventHashPrune(
		[]auditEventHashRecord{third, first, second},
		nil,
		time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Pruned) != 2 ||
		plan.Anchor.ID != second.ID ||
		plan.Retained.ID != third.ID {
		t.Fatalf("unexpected prune plan: %#v", plan)
	}
}

func TestPlanAuditEventHashPruneStopsAtFirstRetainedEvent(t *testing.T) {
	first := testAuditHashRecordAt(t, "ae_1", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	second := testAuditHashRecordAt(t, "ae_2", first.EventHash, time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	third := testAuditHashRecordAt(t, "ae_3", second.EventHash, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))

	plan, err := planAuditEventHashPrune(
		[]auditEventHashRecord{third, second, first},
		nil,
		time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Pruned) != 1 ||
		plan.Anchor.ID != first.ID ||
		plan.Retained.ID != second.ID {
		t.Fatalf("unexpected prune plan: %#v", plan)
	}
}

func TestPlanAuditEventHashPruneRejectsPruningEveryHashedEvent(t *testing.T) {
	first := testAuditHashRecordAt(t, "ae_1", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	second := testAuditHashRecordAt(t, "ae_2", first.EventHash, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))

	_, err := planAuditEventHashPrune(
		[]auditEventHashRecord{first, second},
		nil,
		time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
	)
	if err == nil || !strings.Contains(err.Error(), "every hashed audit event") {
		t.Fatalf("expected prune-all rejection, got %v", err)
	}
}

func TestPlanAuditEventHashPruneAcceptsExistingRetentionAnchor(t *testing.T) {
	anchorHash := strings.Repeat("a", 64)
	first := testAuditHashRecordAt(t, "ae_2", anchorHash, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	second := testAuditHashRecordAt(t, "ae_3", first.EventHash, time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC))

	plan, err := planAuditEventHashPrune(
		[]auditEventHashRecord{first, second},
		map[string]struct{}{anchorHash: {}},
		time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Pruned) != 1 ||
		plan.Anchor.ID != first.ID ||
		plan.Retained.ID != second.ID {
		t.Fatalf("unexpected prune plan: %#v", plan)
	}
}

func testAuditHashRecordAt(
	t *testing.T,
	id string,
	previousHash string,
	createdAt time.Time,
) auditEventHashRecord {
	t.Helper()

	record := testAuditHashRecord(t, id, previousHash)
	record.CreatedAt = createdAt
	hash, err := recomputeAuditEventHash(record)
	if err != nil {
		t.Fatal(err)
	}
	record.EventHash = hash
	return record
}
