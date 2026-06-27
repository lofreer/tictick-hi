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

func TestClaimableLeasePredicateSelectsUnlockedOrExpiredRows(t *testing.T) {
	predicate := claimableLeasePredicate()

	if predicate != "(locked_until IS NULL OR locked_until < now())" {
		t.Fatalf("unexpected predicate %q", predicate)
	}
}

func TestClaimLeaseAssignmentsIncludesHeartbeatAndExtraFieldsForTaskTables(t *testing.T) {
	assignments := claimLeaseAssignments(
		backtestTaskLease,
		"$3",
		"$4",
		"started_at = COALESCE(started_at, now())",
	)

	for _, expected := range []string{
		"locked_by = $3",
		"locked_until = now() + $4::interval",
		"heartbeat_at = now()",
		"started_at = COALESCE(started_at, now())",
		"attempt_count = attempt_count + 1",
		"updated_at = now()",
	} {
		if !strings.Contains(assignments, expected) {
			t.Fatalf("assignments %q missing %q", assignments, expected)
		}
	}
}

func TestClaimLeaseAssignmentsOmitsHeartbeatForNotificationOutbox(t *testing.T) {
	assignments := claimLeaseAssignments(
		notificationOutboxLease,
		"$2",
		"$3",
		"last_attempt_at = now()",
	)

	for _, expected := range []string{
		"locked_by = $2",
		"locked_until = now() + $3::interval",
		"last_attempt_at = now()",
		"attempt_count = attempt_count + 1",
		"updated_at = now()",
	} {
		if !strings.Contains(assignments, expected) {
			t.Fatalf("assignments %q missing %q", assignments, expected)
		}
	}
	if strings.Contains(assignments, "heartbeat_at") {
		t.Fatalf("notification outbox should not reference heartbeat_at: %q", assignments)
	}
}
