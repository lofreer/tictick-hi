package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

const tradingTaskColumns = `
	id, name, type, exchange, account_id, symbol, interval, strategy_id,
	strategy_params::text, intent_policy::text, status,
	COALESCE(locked_by, ''), locked_until, heartbeat_at, started_at,
	finished_at, COALESCE(last_error, ''), attempt_count, created_at, updated_at`

func (store *Store) ListTradingTasks(ctx context.Context) ([]data.TradingTask, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT `+tradingTaskColumns+`
		  FROM trading_tasks
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list trading tasks: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanTradingTask)
}

func (store *Store) CreateTradingTask(
	ctx context.Context,
	task data.CreateTradingTask,
) (data.TradingTask, error) {
	id, err := core.NewPrefixedID("tt")
	if err != nil {
		return data.TradingTask{}, err
	}
	paramsJSON, err := jsonText(task.StrategyParams)
	if err != nil {
		return data.TradingTask{}, err
	}
	policyJSON, err := jsonText(task.IntentPolicy)
	if err != nil {
		return data.TradingTask{}, err
	}

	row := store.pool.QueryRow(ctx, `
		INSERT INTO trading_tasks (
			id, name, type, exchange, account_id, symbol, interval, strategy_id,
			strategy_params, intent_policy
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb)
		RETURNING `+tradingTaskColumns,
		id,
		task.Name,
		task.Type,
		task.Exchange,
		task.AccountID,
		task.Symbol,
		task.Interval,
		task.StrategyID,
		paramsJSON,
		policyJSON,
	)

	created, err := scanTradingTaskRow(row)
	if err != nil {
		return data.TradingTask{}, fmt.Errorf("create trading task: %w", err)
	}
	return created, nil
}

func (store *Store) GetTradingTask(ctx context.Context, id string) (data.TradingTask, error) {
	row := store.pool.QueryRow(ctx, `
		SELECT `+tradingTaskColumns+`
		  FROM trading_tasks
		 WHERE id = $1`, id)

	task, err := scanTradingTaskRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return data.TradingTask{}, data.ErrNotFound
		}
		return data.TradingTask{}, fmt.Errorf("get trading task: %w", err)
	}
	return task, nil
}

func (store *Store) SetTradingTaskStatus(
	ctx context.Context,
	id string,
	status data.TaskStatus,
) (data.TradingTask, error) {
	row := store.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE trading_tasks
		   SET status = $2,
		       %s,
		       started_at = CASE WHEN $2 = $3 THEN COALESCE(started_at, now()) ELSE started_at END,
		       finished_at = CASE WHEN $2 IN ($4, $5, $6) THEN now() ELSE finished_at END,
		       updated_at = now()
		 WHERE id = $1
		RETURNING `+tradingTaskColumns, clearLeaseCaseAssignments(tradingTaskLease, "$2 IN ($4, $5, $6)")),
		id,
		status,
		data.TaskStatusRunning,
		data.TaskStatusPaused,
		data.TaskStatusFailed,
		data.TaskStatusCancelled,
	)

	task, err := scanTradingTaskRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return data.TradingTask{}, data.ErrNotFound
		}
		return data.TradingTask{}, fmt.Errorf("set trading task status: %w", err)
	}
	return task, nil
}

func (store *Store) ListTradingIntents(ctx context.Context, taskID string) ([]data.StrategyIntent, error) {
	return store.listTradingIntents(ctx, taskID)
}

func (store *Store) ListTradingOrders(ctx context.Context, taskID string) ([]data.Order, error) {
	return store.listTradingOrders(ctx, taskID)
}

func (store *Store) ListTradingExecutions(ctx context.Context, taskID string) ([]data.Execution, error) {
	return store.listTradingExecutions(ctx, taskID)
}

func (store *Store) ListTradingPositions(ctx context.Context, taskID string) ([]data.Position, error) {
	return store.listTradingPositions(ctx, taskID)
}

func (store *Store) ListTradingNotifications(ctx context.Context, taskID string) ([]data.Notification, error) {
	return store.listTradingNotifications(ctx, taskID)
}

func (store *Store) ClaimTradingTask(
	ctx context.Context,
	workerID string,
	leaseTTL time.Duration,
) (data.TradingTask, bool, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.TradingTask{}, false, fmt.Errorf("begin claim trading task: %w", err)
	}
	defer tx.Rollback(ctx)

	row, ok, err := claimLeaseRow(
		ctx,
		tx,
		leaseClaimQuery{
			resource: tradingTaskLease,
			where:    "status = $1",
			orderBy:  "updated_at ASC, created_at ASC",
			args: []any{
				data.TaskStatusRunning,
			},
		},
		leaseClaimUpdate{
			resource:  tradingTaskLease,
			workerArg: "$2",
			ttlArg:    "$3",
			extraAssignments: []string{
				"started_at = COALESCE(started_at, now())",
			},
			returningColumns: tradingTaskColumns,
		},
		workerID,
		intervalLiteral(leaseTTL),
	)
	if err != nil {
		return data.TradingTask{}, false, fmt.Errorf("claim trading task: %w", err)
	}
	if !ok {
		return data.TradingTask{}, false, nil
	}
	task, err := scanTradingTaskRow(row)
	if err != nil {
		return data.TradingTask{}, false, fmt.Errorf("update claimed trading task: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return data.TradingTask{}, false, fmt.Errorf("commit claim trading task: %w", err)
	}
	return task, true, nil
}

func (store *Store) HeartbeatTradingTask(
	ctx context.Context,
	taskID string,
	workerID string,
	leaseTTL time.Duration,
) error {
	alive, err := heartbeatLease(
		ctx,
		store.pool,
		tradingTaskLease,
		taskID,
		workerID,
		intervalLiteral(leaseTTL),
		string(data.TaskStatusRunning),
	)
	if err != nil {
		return fmt.Errorf("heartbeat trading task: %w", err)
	}
	if !alive {
		return data.ErrNotFound
	}
	return nil
}

func (store *Store) SaveTradingRunResult(ctx context.Context, result data.TradingRunResult) error {
	return store.saveTradingRunResult(ctx, result)
}

func (store *Store) MarkTradingTaskFailed(ctx context.Context, taskID string, taskErr error) error {
	_, err := store.pool.Exec(ctx, fmt.Sprintf(`
		UPDATE trading_tasks
		   SET status = $2,
		       %s,
		       last_error = $3,
		       updated_at = now()
		 WHERE id = $1`, clearLeaseAssignments(tradingTaskLease)),
		taskID,
		data.TaskStatusFailed,
		taskErr.Error(),
	)
	if err != nil {
		return fmt.Errorf("mark trading task failed: %w", err)
	}
	return nil
}

func (store *Store) ReleaseTradingTask(ctx context.Context, taskID string) error {
	if err := releaseLease(ctx, store.pool, tradingTaskLease, taskID); err != nil {
		return fmt.Errorf("release trading task: %w", err)
	}
	return nil
}
