package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestNormalizeAuditMetadataBoundsText(t *testing.T) {
	longKey := strings.Repeat("k", maxAuditMetadataKeyLength+10)
	longValue := "  " + strings.Repeat("v", maxAuditMetadataValueLength+10) + "  "

	metadata := normalizeAuditMetadata(map[string]string{
		" enabled ": " false ",
		"   ":       "ignored",
		longKey:     longValue,
	})

	if len(metadata) != 2 {
		t.Fatalf("metadata length = %d, metadata = %#v", len(metadata), metadata)
	}
	if metadata["enabled"] != "false" {
		t.Fatalf("enabled metadata = %q", metadata["enabled"])
	}
	boundedKey := boundedAuditMetadataText(longKey, maxAuditMetadataKeyLength)
	if len([]rune(boundedKey)) != maxAuditMetadataKeyLength ||
		len([]rune(metadata[boundedKey])) != maxAuditMetadataValueLength {
		t.Fatalf("bounded metadata key/value lengths = %d/%d", len([]rune(boundedKey)), len([]rune(metadata[boundedKey])))
	}
	if strings.Contains(metadata[boundedKey], " ") {
		t.Fatalf("bounded metadata value was not trimmed: %q", metadata[boundedKey])
	}
}

func TestNormalizeAuditMetadataLimitsEntriesDeterministically(t *testing.T) {
	metadata := make(map[string]string, maxAuditMetadataEntries+5)
	for index := range maxAuditMetadataEntries + 5 {
		metadata[fmt.Sprintf("key_%02d", index)] = "value"
	}

	normalized := normalizeAuditMetadata(metadata)

	if len(normalized) != maxAuditMetadataEntries {
		t.Fatalf("metadata length = %d, want %d", len(normalized), maxAuditMetadataEntries)
	}
	if _, exists := normalized["key_00"]; !exists {
		t.Fatalf("metadata missing first sorted key: %#v", normalized)
	}
	if _, exists := normalized["key_19"]; !exists {
		t.Fatalf("metadata missing last retained key: %#v", normalized)
	}
	if _, exists := normalized["key_20"]; exists {
		t.Fatalf("metadata retained key past limit: %#v", normalized)
	}
}

func TestAuditEventHashIsDeterministicAndChainsPreviousHash(t *testing.T) {
	input := auditEventHashInput{
		ID:              "ae_hash_1",
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
		CreatedAt:       time.Date(2026, 7, 7, 12, 0, 0, 123456000, time.UTC),
	}

	firstHash, err := auditEventHash(input)
	if err != nil {
		t.Fatal(err)
	}
	repeatedHash, err := auditEventHash(input)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash != repeatedHash {
		t.Fatalf("audit event hash is not deterministic: %s != %s", firstHash, repeatedHash)
	}
	if _, err := hex.DecodeString(firstHash); err != nil || len(firstHash) != 64 {
		t.Fatalf("audit event hash is not 64-char hex: %q err=%v", firstHash, err)
	}

	input.PreviousHash = firstHash
	chainedHash, err := auditEventHash(input)
	if err != nil {
		t.Fatal(err)
	}
	if chainedHash == firstHash {
		t.Fatal("audit event hash did not change when previous hash changed")
	}

	input.Metadata = []byte(`{"enabled":"true"}`)
	changedHash, err := auditEventHash(input)
	if err != nil {
		t.Fatal(err)
	}
	if changedHash == chainedHash {
		t.Fatal("audit event hash did not change when metadata changed")
	}
}

func TestIntegrationAuditEventsRoundTrip(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: "audit_" + suffix,
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM audit_events WHERE actor_operator_id = $1`, operator.ID)
	})

	event, err := store.RecordAuditEvent(ctx, data.CreateAuditEvent{
		ActorOperatorID: operator.ID,
		ActorUsername:   operator.Username,
		Action:          "operator.disable",
		ResourceType:    "operator",
		ResourceID:      operator.ID,
		Outcome:         "success",
		RequestMethod:   "POST",
		RequestPath:     "/api/system/operators/" + operator.ID + "/disable",
		RemoteAddr:      "127.0.0.1",
		UserAgent:       "integration-test",
		Metadata: map[string]string{
			"enabled": "false",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.ID == "" || event.Metadata["enabled"] != "false" {
		t.Fatalf("unexpected created audit event: %#v", event)
	}
	if _, err := hex.DecodeString(event.EventHash); err != nil || len(event.EventHash) != 64 {
		t.Fatalf("unexpected event hash: %q err=%v", event.EventHash, err)
	}

	secondEvent, err := store.RecordAuditEvent(ctx, data.CreateAuditEvent{
		ActorOperatorID: operator.ID,
		ActorUsername:   operator.Username,
		Action:          "operator.enable",
		ResourceType:    "operator",
		ResourceID:      operator.ID,
		Outcome:         "success",
		RequestMethod:   "POST",
		RequestPath:     "/api/system/operators/" + operator.ID + "/enable",
		RemoteAddr:      "127.0.0.1",
		UserAgent:       "integration-test",
		Metadata: map[string]string{
			"enabled": "true",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if secondEvent.PreviousHash != event.EventHash {
		t.Fatalf("second audit previous hash = %q, want %q", secondEvent.PreviousHash, event.EventHash)
	}
	if secondEvent.EventHash == "" || secondEvent.EventHash == event.EventHash {
		t.Fatalf("unexpected second event hash: %#v", secondEvent)
	}

	events, err := store.ListAuditEvents(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	listedEvent, ok := findAuditEvent(events, event.ID)
	if !ok {
		t.Fatalf("unexpected listed audit events: %#v", events)
	}
	if listedEvent.ActorOperatorID != operator.ID ||
		listedEvent.Action != "operator.disable" ||
		listedEvent.ResourceID != operator.ID ||
		listedEvent.Metadata["enabled"] != "false" ||
		listedEvent.EventHash != event.EventHash {
		t.Fatalf("unexpected audit event round trip: %#v", listedEvent)
	}
}

func TestIntegrationAuditEventConstraintsRejectInvalidOutcome(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := store.pool.Exec(ctx, `
			INSERT INTO audit_events (id, action, resource_type, outcome, event_hash)
			VALUES ($1, 'operator.disable', 'operator', 'maybe', $2)`,
		integrationID("ae_bad_outcome"),
		auditTestHash("bad_outcome"),
	)
	if err == nil {
		t.Fatal("expected audit_events_outcome_check violation")
	}
	assertDatabaseConstraintError(t, err, "audit_events_outcome_check")
}

func TestIntegrationAuditEventConstraintsRejectBlankRequiredText(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := store.pool.Exec(ctx, `
			INSERT INTO audit_events (id, action, resource_type, outcome, event_hash)
			VALUES ($1, '   ', 'operator', 'success', $2)`,
		integrationID("ae_blank_action"),
		auditTestHash("blank_action"),
	)
	if err == nil {
		t.Fatal("expected audit_events_trimmed_required_text_check violation")
	}
	assertDatabaseConstraintError(t, err, "audit_events_trimmed_required_text_check")
}

func findAuditEvent(events []data.AuditEvent, id string) (data.AuditEvent, bool) {
	for _, event := range events {
		if event.ID == id {
			return event, true
		}
	}
	return data.AuditEvent{}, false
}

func auditTestHash(seed string) string {
	sum := sha256.Sum256([]byte(integrationID(seed)))
	return hex.EncodeToString(sum[:])
}
