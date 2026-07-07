package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

const backtestTaskColumns = `
	id, name, exchange, symbol, interval, start_time, end_time,
	COALESCE(request_id, ''), strategy_id, strategy_params::text, initial_balance::text,
	fee_bps::text, slippage_bps::text, trigger_mode, status,
	started_at, finished_at, COALESCE(last_error, ''), attempt_count,
	result_summary::text, created_at, updated_at`

func (store *Store) ListBacktestTasks(ctx context.Context) ([]data.BacktestTask, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT `+backtestTaskColumns+`
		  FROM backtest_tasks
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list backtest tasks: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanBacktestTask)
}

func (store *Store) CreateBacktestTask(
	ctx context.Context,
	task data.CreateBacktestTask,
) (data.BacktestTask, error) {
	id, err := core.NewPrefixedID("bt")
	if err != nil {
		return data.BacktestTask{}, err
	}
	paramsJSON, err := jsonText(task.StrategyParams)
	if err != nil {
		return data.BacktestTask{}, err
	}

	row := store.pool.QueryRow(ctx, `
		INSERT INTO backtest_tasks (
			id, name, exchange, symbol, interval, start_time, end_time,
			strategy_id, strategy_params, initial_balance, fee_bps,
			slippage_bps, trigger_mode, request_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb,
		        $10::numeric, $11::numeric, $12::numeric, $13, NULLIF($14, ''))
		RETURNING `+backtestTaskColumns,
		id,
		task.Name,
		task.Exchange,
		task.Symbol,
		task.Interval,
		task.StartTime,
		task.EndTime,
		task.StrategyID,
		paramsJSON,
		task.InitialBalance,
		task.FeeBps,
		task.SlippageBps,
		task.TriggerMode,
		task.RequestID,
	)

	created, err := scanBacktestTaskRow(row)
	if err != nil {
		return data.BacktestTask{}, fmt.Errorf("create backtest task: %w", err)
	}
	return created, nil
}

func (store *Store) GetBacktestTask(ctx context.Context, id string) (data.BacktestTask, error) {
	row := store.pool.QueryRow(ctx, `
		SELECT `+backtestTaskColumns+`
		  FROM backtest_tasks
		 WHERE id = $1`, id)

	task, err := scanBacktestTaskRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return data.BacktestTask{}, data.ErrNotFound
		}
		return data.BacktestTask{}, fmt.Errorf("get backtest task: %w", err)
	}
	return task, nil
}

func (store *Store) ListBacktestOrders(ctx context.Context, backtestID string) ([]data.BacktestOrder, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, backtest_id, COALESCE(intent_id, ''), side,
		       price::text, quantity::text, status, occurred_at
		  FROM backtest_orders
		 WHERE backtest_id = $1
		 ORDER BY occurred_at ASC`, backtestID)
	if err != nil {
		return nil, fmt.Errorf("list backtest orders: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanBacktestOrder)
}

func (store *Store) ListBacktestIntents(ctx context.Context, backtestID string) ([]data.StrategyIntent, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, task_id, task_type, strategy_id, intent_type, idempotency_key,
		       payload::text, policy, status, created_at
		  FROM strategy_intents
		 WHERE task_id = $1
		   AND task_type = 'backtest'
		 ORDER BY created_at ASC`, backtestID)
	if err != nil {
		return nil, fmt.Errorf("list backtest intents: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanStrategyIntent)
}

func (store *Store) ClaimBacktestTask(
	ctx context.Context,
	workerID string,
	leaseTTL time.Duration,
) (data.BacktestTask, bool, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.BacktestTask{}, false, fmt.Errorf("begin claim backtest task: %w", err)
	}
	defer tx.Rollback(ctx)

	row, ok, err := claimLeaseRow(
		ctx,
		tx,
		leaseClaimQuery{
			resource: backtestTaskLease,
			where:    "status = $1",
			orderBy:  "created_at ASC",
			args: []any{
				data.TaskStatusPending,
			},
		},
		leaseClaimUpdate{
			resource:         backtestTaskLease,
			statusAssignment: "status = $2",
			workerArg:        "$3",
			ttlArg:           "$4",
			extraAssignments: []string{
				"started_at = COALESCE(started_at, now())",
			},
			returningColumns: backtestTaskColumns,
		},
		data.TaskStatusRunning,
		workerID,
		intervalLiteral(leaseTTL),
	)
	if err != nil {
		return data.BacktestTask{}, false, fmt.Errorf("claim backtest task: %w", err)
	}
	if !ok {
		return data.BacktestTask{}, false, nil
	}
	task, err := scanBacktestTaskRow(row)
	if err != nil {
		return data.BacktestTask{}, false, fmt.Errorf("update claimed backtest task: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return data.BacktestTask{}, false, fmt.Errorf("commit claim backtest task: %w", err)
	}
	return task, true, nil
}

func (store *Store) HeartbeatBacktestTask(
	ctx context.Context,
	taskID string,
	workerID string,
	leaseTTL time.Duration,
) error {
	alive, err := heartbeatLease(
		ctx,
		store.pool,
		backtestTaskLease,
		taskID,
		workerID,
		intervalLiteral(leaseTTL),
		string(data.TaskStatusRunning),
	)
	if err != nil {
		return fmt.Errorf("heartbeat backtest task: %w", err)
	}
	if !alive {
		return fmt.Errorf("heartbeat backtest task: lease lost for %s", taskID)
	}
	return nil
}

func (store *Store) SaveBacktestResult(ctx context.Context, result data.BacktestResult) error {
	summaryJSON, err := jsonText(result.ResultSummary)
	if err != nil {
		return err
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save backtest result: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM backtest_orders WHERE backtest_id = $1`, result.TaskID); err != nil {
		return fmt.Errorf("delete backtest orders: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM strategy_intents
		 WHERE task_id = $1
		   AND task_type = 'backtest'`, result.TaskID); err != nil {
		return fmt.Errorf("delete backtest intents: %w", err)
	}
	for _, intent := range result.Intents {
		payloadJSON, err := jsonText(intent.Payload)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO strategy_intents (
				id, task_id, task_type, strategy_id, intent_type, idempotency_key,
				payload, policy, status, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)
			ON CONFLICT (task_id, idempotency_key)
			DO UPDATE SET payload = EXCLUDED.payload, policy = EXCLUDED.policy, status = EXCLUDED.status`,
			intent.ID,
			result.TaskID,
			intent.TaskType,
			intent.StrategyID,
			intent.IntentType,
			intent.IdempotencyKey,
			payloadJSON,
			intent.Policy,
			intent.Status,
			intent.CreatedAt,
		); err != nil {
			return fmt.Errorf("upsert backtest intent: %w", err)
		}
	}

	for _, order := range result.Orders {
		if _, err := tx.Exec(ctx, `
			INSERT INTO backtest_orders (
				id, backtest_id, intent_id, side, price, quantity, status, occurred_at
			)
			VALUES ($1, $2, $3, $4, $5::numeric, $6::numeric, $7, $8)`,
			order.ID,
			result.TaskID,
			order.IntentID,
			order.Side,
			order.Price,
			order.Quantity,
			order.Status,
			order.OccurredAt,
		); err != nil {
			return fmt.Errorf("insert backtest order: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, leaseTransitionUpdateSQL(leaseTransitionUpdate{
		resource: backtestTaskLease,
		assignments: []string{
			"status = $2",
			"result_summary = $3::jsonb",
			"finished_at = now()",
			"last_error = NULL",
		},
		where: "id = $1",
	}),
		result.TaskID,
		data.TaskStatusSucceeded,
		summaryJSON,
	); err != nil {
		return fmt.Errorf("update backtest result: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit backtest result: %w", err)
	}
	return nil
}

func (store *Store) MarkBacktestFailed(ctx context.Context, taskID string, taskErr error) error {
	_, err := store.pool.Exec(ctx, leaseTransitionUpdateSQL(leaseTransitionUpdate{
		resource: backtestTaskLease,
		assignments: []string{
			"status = $2",
			"last_error = $3",
			"finished_at = now()",
		},
		where: "id = $1",
	}),
		taskID,
		data.TaskStatusFailed,
		taskErr.Error(),
	)
	if err != nil {
		return fmt.Errorf("mark backtest failed: %w", err)
	}
	return nil
}

func (store *Store) ReleaseBacktestTask(ctx context.Context, taskID string) error {
	_, err := store.pool.Exec(ctx, leaseTransitionUpdateSQL(leaseTransitionUpdate{
		resource: backtestTaskLease,
		assignments: []string{
			"status = $2",
		},
		where: "id = $1\n\t\t\t   AND status = $3",
	}),
		taskID,
		data.TaskStatusPending,
		data.TaskStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("release backtest task: %w", err)
	}
	return nil
}

func scanBacktestTask(row pgx.CollectableRow) (data.BacktestTask, error) {
	return scanBacktestTaskRow(row)
}

func scanBacktestTaskRow(row rowScanner) (data.BacktestTask, error) {
	var (
		task               data.BacktestTask
		strategyParamsJSON string
		resultSummaryJSON  string
	)
	err := row.Scan(
		&task.ID,
		&task.Name,
		&task.Exchange,
		&task.Symbol,
		&task.Interval,
		&task.StartTime,
		&task.EndTime,
		&task.RequestID,
		&task.StrategyID,
		&strategyParamsJSON,
		&task.InitialBalance,
		&task.FeeBps,
		&task.SlippageBps,
		&task.TriggerMode,
		&task.Status,
		&task.StartedAt,
		&task.FinishedAt,
		&task.LastError,
		&task.AttemptCount,
		&resultSummaryJSON,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return data.BacktestTask{}, err
	}

	strategyParams, err := jsonMap(strategyParamsJSON)
	if err != nil {
		return data.BacktestTask{}, err
	}
	resultSummary, err := jsonMap(resultSummaryJSON)
	if err != nil {
		return data.BacktestTask{}, err
	}
	task.StrategyParams = strategyParams
	task.ResultSummary = resultSummary
	return task, nil
}

func scanBacktestOrder(row pgx.CollectableRow) (data.BacktestOrder, error) {
	var order data.BacktestOrder
	err := row.Scan(
		&order.ID,
		&order.BacktestID,
		&order.IntentID,
		&order.Side,
		&order.Price,
		&order.Quantity,
		&order.Status,
		&order.OccurredAt,
	)
	return order, err
}

func jsonText(value map[string]any) (string, error) {
	if value == nil {
		value = map[string]any{}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode json map: %w", err)
	}
	return string(encoded), nil
}

func jsonMap(value string) (map[string]any, error) {
	if value == "" {
		return map[string]any{}, nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil, fmt.Errorf("decode json map: %w", err)
	}
	return decoded, nil
}
