package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
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

type leaseQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type leaseClaimQuery struct {
	resource leaseResource
	where    string
	orderBy  string
	args     []any
}

type leaseClaimUpdate struct {
	resource         leaseResource
	statusAssignment string
	workerArg        string
	ttlArg           string
	extraAssignments []string
	returningColumns string
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

func claimableLeasePredicate() string {
	return "(locked_until IS NULL OR locked_until < now())"
}

func claimLeaseIDSQL(query leaseClaimQuery) string {
	return fmt.Sprintf(`
			SELECT %s
			  FROM %s
			 WHERE %s
			   AND %s
			 ORDER BY %s
			 LIMIT 1
			 FOR UPDATE SKIP LOCKED`,
		query.resource.keyColumn,
		query.resource.table,
		query.where,
		claimableLeasePredicate(),
		query.orderBy,
	)
}

func claimLeaseID(ctx context.Context, queryer leaseQueryer, query leaseClaimQuery) (string, bool, error) {
	var id string
	if err := queryer.QueryRow(ctx, claimLeaseIDSQL(query), query.args...).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return id, true, nil
}

func claimLeaseUpdateSQL(update leaseClaimUpdate) string {
	assignments := claimLeaseAssignments(
		update.resource,
		update.workerArg,
		update.ttlArg,
		update.extraAssignments...,
	)
	if update.statusAssignment != "" {
		assignments = update.statusAssignment + ",\n		       " + assignments
	}
	return fmt.Sprintf(`
			UPDATE %s
			   SET %s
			 WHERE %s = $1
			RETURNING %s`,
		update.resource.table,
		assignments,
		update.resource.keyColumn,
		update.returningColumns,
	)
}

func claimLeaseRow(
	ctx context.Context,
	queryer leaseQueryer,
	query leaseClaimQuery,
	update leaseClaimUpdate,
	updateArgs ...any,
) (pgx.Row, bool, error) {
	id, ok, err := claimLeaseID(ctx, queryer, query)
	if err != nil || !ok {
		return nil, ok, err
	}
	args := make([]any, 0, len(updateArgs)+1)
	args = append(args, id)
	args = append(args, updateArgs...)
	return queryer.QueryRow(ctx, claimLeaseUpdateSQL(update), args...), true, nil
}

func claimLeaseAssignments(resource leaseResource, workerArg string, ttlArg string, extraAssignments ...string) string {
	assignments := []string{
		"locked_by = " + workerArg,
		"locked_until = now() + " + ttlArg + "::interval",
	}
	if resource.hasHeartbeat {
		assignments = append(assignments, "heartbeat_at = now()")
	}
	assignments = append(assignments, extraAssignments...)
	assignments = append(
		assignments,
		"attempt_count = attempt_count + 1",
		"updated_at = now()",
	)
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

func heartbeatLease(
	ctx context.Context,
	exec leaseExec,
	resource leaseResource,
	id string,
	workerID string,
	leaseTTLSeconds string,
	status string,
) (bool, error) {
	if !resource.hasHeartbeat {
		return false, fmt.Errorf("lease resource %s does not support heartbeat", resource.table)
	}
	commandTag, err := exec.Exec(ctx, fmt.Sprintf(`
			UPDATE %s
			   SET heartbeat_at = now(),
			       locked_until = now() + $3::interval,
			       updated_at = now()
			 WHERE %s = $1
			   AND locked_by = $2
			   AND status = $4`,
		resource.table,
		resource.keyColumn,
	),
		id,
		workerID,
		leaseTTLSeconds,
		status,
	)
	if err != nil {
		return false, err
	}
	return commandTag.RowsAffected() > 0, nil
}
