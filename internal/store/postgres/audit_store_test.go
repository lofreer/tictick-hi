package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

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

	events, err := store.ListAuditEvents(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("unexpected listed audit events: %#v", events)
	}
	if events[0].ActorOperatorID != operator.ID ||
		events[0].Action != "operator.disable" ||
		events[0].ResourceID != operator.ID ||
		events[0].Metadata["enabled"] != "false" {
		t.Fatalf("unexpected audit event round trip: %#v", events[0])
	}
}

func TestIntegrationAuditEventConstraintsRejectInvalidOutcome(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := store.pool.Exec(ctx, `
		INSERT INTO audit_events (id, action, resource_type, outcome)
		VALUES ('ae_bad_outcome', 'operator.disable', 'operator', 'maybe')`)
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
		INSERT INTO audit_events (id, action, resource_type, outcome)
		VALUES ($1, '   ', 'operator', 'success')`,
		integrationID("ae_blank_action"),
	)
	if err == nil {
		t.Fatal("expected audit_events_trimmed_required_text_check violation")
	}
	assertDatabaseConstraintError(t, err, "audit_events_trimmed_required_text_check")
}
