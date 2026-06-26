package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

const backtestTaskColumns = `
	id, name, exchange, symbol, interval, start_time, end_time,
	strategy_id, strategy_params::text, initial_balance::text,
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
			slippage_bps, trigger_mode
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb,
		        $10::numeric, $11::numeric, $12::numeric, $13)
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
