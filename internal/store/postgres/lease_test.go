package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestClaimLeaseIDSQLUsesResourceCandidateAndLeasePredicate(t *testing.T) {
	query := leaseClaimQuery{
		resource: tradingTaskLease,
		where:    "status = $1",
		orderBy:  "updated_at ASC, created_at ASC",
	}

	sql := claimLeaseIDSQL(query)

	for _, expected := range []string{
		"SELECT id",
		"FROM trading_tasks",
		"WHERE status = $1",
		"AND (locked_until IS NULL OR locked_until < now())",
		"ORDER BY updated_at ASC, created_at ASC",
		"FOR UPDATE SKIP LOCKED",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("claim sql %q missing %q", sql, expected)
		}
	}
}

func TestClaimLeaseIDReportsNoRowsAsEmpty(t *testing.T) {
	queryer := &captureLeaseQueryer{err: pgx.ErrNoRows}

	id, ok, err := claimLeaseID(context.Background(), queryer, leaseClaimQuery{
		resource: dataSyncTaskLease,
		where:    "status = $1",
		orderBy:  "created_at ASC",
		args:     []any{"pending"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no claim")
	}
	if id != "" {
		t.Fatalf("id = %q, want empty", id)
	}
}

func TestClaimLeaseIDPropagatesQueryErrors(t *testing.T) {
	queryErr := errors.New("database unavailable")
	queryer := &captureLeaseQueryer{err: queryErr}

	id, ok, err := claimLeaseID(context.Background(), queryer, leaseClaimQuery{
		resource: backtestTaskLease,
		where:    "status = $1",
		orderBy:  "created_at ASC",
		args:     []any{"pending"},
	})
	if !errors.Is(err, queryErr) {
		t.Fatalf("err = %v, want %v", err, queryErr)
	}
	if ok {
		t.Fatal("query errors must not be reported as claimed")
	}
	if id != "" {
		t.Fatalf("id = %q, want empty", id)
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

func TestHeartbeatLeaseUpdatesTaskLeaseFields(t *testing.T) {
	exec := &captureLeaseExec{tag: pgconn.NewCommandTag("UPDATE 1")}

	alive, err := heartbeatLease(
		context.Background(),
		exec,
		tradingTaskLease,
		"tt_1",
		"worker-1",
		"30.000000 seconds",
		"running",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !alive {
		t.Fatal("expected heartbeat to keep lease alive")
	}
	for _, expected := range []string{
		"UPDATE trading_tasks",
		"heartbeat_at = now()",
		"locked_until = now() + $3::interval",
		"WHERE id = $1",
		"AND locked_by = $2",
		"AND status = $4",
	} {
		if !strings.Contains(exec.sql, expected) {
			t.Fatalf("heartbeat sql %q missing %q", exec.sql, expected)
		}
	}
	expectedArgs := []any{"tt_1", "worker-1", "30.000000 seconds", "running"}
	if len(exec.args) != len(expectedArgs) {
		t.Fatalf("args = %#v", exec.args)
	}
	for index, expected := range expectedArgs {
		if exec.args[index] != expected {
			t.Fatalf("arg %d = %#v, want %#v", index, exec.args[index], expected)
		}
	}
}

func TestHeartbeatLeaseReportsLostLeaseWhenNoRowsAffected(t *testing.T) {
	exec := &captureLeaseExec{tag: pgconn.NewCommandTag("UPDATE 0")}

	alive, err := heartbeatLease(
		context.Background(),
		exec,
		dataSyncTaskLease,
		"dst_1",
		"worker-1",
		"30.000000 seconds",
		"running",
	)
	if err != nil {
		t.Fatal(err)
	}
	if alive {
		t.Fatal("expected heartbeat to report lost lease")
	}
}

func TestHeartbeatLeaseRejectsResourcesWithoutHeartbeat(t *testing.T) {
	exec := &captureLeaseExec{tag: pgconn.NewCommandTag("UPDATE 1")}

	alive, err := heartbeatLease(
		context.Background(),
		exec,
		notificationOutboxLease,
		"no_1",
		"worker-1",
		"30.000000 seconds",
		"running",
	)
	if err == nil {
		t.Fatal("expected heartbeat error")
	}
	if alive {
		t.Fatal("notification outbox should not heartbeat")
	}
	if exec.sql != "" {
		t.Fatalf("heartbeat should not execute for notification outbox: %q", exec.sql)
	}
}

type captureLeaseExec struct {
	sql  string
	args []any
	tag  pgconn.CommandTag
	err  error
}

func (exec *captureLeaseExec) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	exec.sql = sql
	exec.args = arguments
	return exec.tag, exec.err
}

type captureLeaseQueryer struct {
	sql  string
	args []any
	id   string
	err  error
}

func (queryer *captureLeaseQueryer) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	queryer.sql = sql
	queryer.args = args
	return captureLeaseRow{id: queryer.id, err: queryer.err}
}

type captureLeaseRow struct {
	id  string
	err error
}

func (row captureLeaseRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*(dest[0].(*string)) = row.id
	return nil
}
