package postgres

import (
	"strings"
	"testing"
)

func TestClearLeaseAssignmentsIncludesHeartbeatForTaskTables(t *testing.T) {
	assignments := clearLeaseAssignments(dataSyncTaskLease)

	for _, expected := range []string{
		"locked_by = NULL",
		"locked_until = NULL",
		"heartbeat_at = NULL",
	} {
		if !strings.Contains(assignments, expected) {
			t.Fatalf("assignments %q missing %q", assignments, expected)
		}
	}
}

func TestClearLeaseAssignmentsOmitsHeartbeatForNotificationOutbox(t *testing.T) {
	assignments := clearLeaseAssignments(notificationOutboxLease)

	if !strings.Contains(assignments, "locked_by = NULL") ||
		!strings.Contains(assignments, "locked_until = NULL") {
		t.Fatalf("assignments %q missing notification lease fields", assignments)
	}
	if strings.Contains(assignments, "heartbeat_at") {
		t.Fatalf("notification outbox should not reference heartbeat_at: %q", assignments)
	}
}

func TestClearLeaseCaseAssignmentsKeepsLeaseWhenConditionIsFalse(t *testing.T) {
	assignments := clearLeaseCaseAssignments(tradingTaskLease, "$2 IN ($4, $5, $6)")

	for _, expected := range []string{
		"locked_by = CASE WHEN $2 IN ($4, $5, $6) THEN NULL ELSE locked_by END",
		"locked_until = CASE WHEN $2 IN ($4, $5, $6) THEN NULL ELSE locked_until END",
		"heartbeat_at = CASE WHEN $2 IN ($4, $5, $6) THEN NULL ELSE heartbeat_at END",
	} {
		if !strings.Contains(assignments, expected) {
			t.Fatalf("assignments %q missing %q", assignments, expected)
		}
	}
}
