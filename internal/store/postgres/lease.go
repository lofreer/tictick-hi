package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type leaseResource struct {
	table        string
	keyColumn    string
	hasHeartbeat bool
}

var (
	dataSyncTaskLease       = leaseResource{table: "data_sync_tasks", keyColumn: "id", hasHeartbeat: true}
	backtestTaskLease       = leaseResource{table: "backtest_tasks", keyColumn: "id", hasHeartbeat: true}
	tradingTaskLease        = leaseResource{table: "trading_tasks", keyColumn: "id", hasHeartbeat: true}
	notificationOutboxLease = leaseResource{table: "notification_outbox", keyColumn: "id"}
)

type leaseExec interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func clearLeaseAssignments(resource leaseResource) string {
	assignments := []string{
		"locked_by = NULL",
		"locked_until = NULL",
	}
	if resource.hasHeartbeat {
		assignments = append(assignments, "heartbeat_at = NULL")
	}
	return strings.Join(assignments, ",\n		       ")
}

func clearLeaseCaseAssignments(resource leaseResource, condition string) string {
	assignments := []string{
		"locked_by = CASE WHEN " + condition + " THEN NULL ELSE locked_by END",
		"locked_until = CASE WHEN " + condition + " THEN NULL ELSE locked_until END",
	}
	if resource.hasHeartbeat {
		assignments = append(
			assignments,
			"heartbeat_at = CASE WHEN "+condition+" THEN NULL ELSE heartbeat_at END",
		)
	}
	return strings.Join(assignments, ",\n		       ")
}

func releaseLease(ctx context.Context, exec leaseExec, resource leaseResource, id string) error {
	_, err := exec.Exec(ctx, fmt.Sprintf(`
		UPDATE %s
		   SET %s,
		       updated_at = now()
		 WHERE %s = $1`,
		resource.table,
		clearLeaseAssignments(resource),
		resource.keyColumn,
	), id)
	return err
}
