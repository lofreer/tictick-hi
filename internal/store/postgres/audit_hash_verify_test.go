package postgres

import (
	"strings"
	"testing"
	"time"
)

func TestVerifyAuditEventHashRecordsAcceptsValidChain(t *testing.T) {
	first := testAuditHashRecord(t, "ae_1", "")
	second := testAuditHashRecord(t, "ae_2", first.EventHash)

	result, err := verifyAuditEventHashRecords(
		[]auditEventHashRecord{first, second},
		1,
		nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "warning" ||
		result.CheckedCount != 2 ||
		result.SkippedCount != 1 ||
		result.RootCount != 1 ||
		result.TailCount != 1 ||
		result.BrokenEventID != "" {
		t.Fatalf("verification result = %#v", result)
	}
}

func TestVerifyAuditEventHashRecordsAcceptsRetentionAnchor(t *testing.T) {
	anchorHash := strings.Repeat("a", 64)
	record := testAuditHashRecord(t, "ae_2", anchorHash)

	result, err := verifyAuditEventHashRecords(
		[]auditEventHashRecord{record},
		0,
		map[string]struct{}{anchorHash: {}},
		time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" ||
		result.CheckedCount != 1 ||
		result.RootCount != 1 ||
		result.TailCount != 1 ||
		result.BrokenEventID != "" {
		t.Fatalf("verification result = %#v", result)
	}
}

func TestVerifyAuditEventHashRecordsDetectsHashMismatch(t *testing.T) {
	record := testAuditHashRecord(t, "ae_1", "")
	record.Action = "operator.enable"

	result, err := verifyAuditEventHashRecords([]auditEventHashRecord{record}, 0, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "failure" ||
		result.BrokenEventID != "ae_1" ||
		!strings.Contains(result.Message, "mismatch") {
		t.Fatalf("verification result = %#v", result)
	}
}

func TestVerifyAuditEventHashRecordsDetectsMissingPreviousHash(t *testing.T) {
	record := testAuditHashRecord(t, "ae_2", strings.Repeat("a", 64))

	result, err := verifyAuditEventHashRecords([]auditEventHashRecord{record}, 0, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "failure" ||
		result.BrokenEventID != "ae_2" ||
		!strings.Contains(result.Message, "previous hash") {
		t.Fatalf("verification result = %#v", result)
	}
}

func TestVerifyAuditEventHashRecordsReportsMalformedMetadata(t *testing.T) {
	record := testAuditHashRecord(t, "ae_1", "")
	record.Metadata = []byte(`{"enabled":false}`)

	result, err := verifyAuditEventHashRecords([]auditEventHashRecord{record}, 0, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "failure" ||
		result.BrokenEventID != "ae_1" ||
		!strings.Contains(result.Message, "metadata") {
		t.Fatalf("verification result = %#v", result)
	}
}

func testAuditHashRecord(t *testing.T, id string, previousHash string) auditEventHashRecord {
	t.Helper()

	record := auditEventHashRecord{
		ID:              id,
		ActorOperatorID: "op_admin",
		ActorUsername:   "admin",
		Action:          "operator.disable",
		ResourceType:    "operator",
		ResourceID:      "op_target",
		Outcome:         "success",
		RequestMethod:   "POST",
		RequestPath:     "/api/system/operators/op_target/disable",
		RemoteAddr:      "127.0.0.1",
		UserAgent:       "audit-test",
		Metadata:        []byte(`{"enabled":"false"}`),
		PreviousHash:    previousHash,
		CreatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	hash, err := recomputeAuditEventHash(record)
	if err != nil {
		t.Fatal(err)
	}
	record.EventHash = hash
	return record
}
