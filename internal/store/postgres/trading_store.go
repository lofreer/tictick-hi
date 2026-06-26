package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

const tradingTaskColumns = `
	id, name, type, exchange, account_id, symbol, strategy_id,
	strategy_params::text, intent_policy::text, status, started_at,
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
			id, name, type, exchange, account_id, symbol, strategy_id,
			strategy_params, intent_policy
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb)
		RETURNING `+tradingTaskColumns,
		id,
		task.Name,
		task.Type,
		task.Exchange,
		task.AccountID,
		task.Symbol,
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
	row := store.pool.QueryRow(ctx, `
		UPDATE trading_tasks
		   SET status = $2,
		       started_at = CASE WHEN $2 = $3 THEN COALESCE(started_at, now()) ELSE started_at END,
		       finished_at = CASE WHEN $2 IN ($4, $5, $6) THEN now() ELSE finished_at END,
		       updated_at = now()
		 WHERE id = $1
		RETURNING `+tradingTaskColumns,
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
	rows, err := store.pool.Query(ctx, `
		SELECT id, task_id, task_type, strategy_id, intent_type, idempotency_key,
		       payload::text, policy, status, created_at
		  FROM strategy_intents
		 WHERE task_id = $1
		 ORDER BY created_at DESC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list trading intents: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanStrategyIntent)
}

func (store *Store) ListTradingOrders(ctx context.Context, taskID string) ([]data.Order, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, task_id, task_type, COALESCE(intent_id, ''), idempotency_key,
		       exchange, account_id, symbol, side, order_type, price::text, quantity::text,
		       status, COALESCE(exchange_order_id, ''), exchange_response_summary::text,
		       COALESCE(last_error, ''), created_at, updated_at
		  FROM orders
		 WHERE task_id = $1
		 ORDER BY created_at DESC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list trading orders: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanOrder)
}

func (store *Store) ListTradingNotifications(ctx context.Context, taskID string) ([]data.Notification, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, COALESCE(intent_id, ''), channel, title, body, status,
		       COALESCE(error, ''), created_at, sent_at
		  FROM notifications
		 WHERE task_id = $1
		 ORDER BY created_at DESC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list trading notifications: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanNotification)
}

func scanTradingTask(row pgx.CollectableRow) (data.TradingTask, error) {
	return scanTradingTaskRow(row)
}

func scanTradingTaskRow(row rowScanner) (data.TradingTask, error) {
	var (
		task               data.TradingTask
		strategyParamsJSON string
		intentPolicyJSON   string
	)
	err := row.Scan(
		&task.ID,
		&task.Name,
		&task.Type,
		&task.Exchange,
		&task.AccountID,
		&task.Symbol,
		&task.StrategyID,
		&strategyParamsJSON,
		&intentPolicyJSON,
		&task.Status,
		&task.StartedAt,
		&task.FinishedAt,
		&task.LastError,
		&task.AttemptCount,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return data.TradingTask{}, err
	}

	strategyParams, err := jsonMap(strategyParamsJSON)
	if err != nil {
		return data.TradingTask{}, err
	}
	intentPolicy, err := jsonMap(intentPolicyJSON)
	if err != nil {
		return data.TradingTask{}, err
	}
	task.StrategyParams = strategyParams
	task.IntentPolicy = intentPolicy
	return task, nil
}

func scanStrategyIntent(row pgx.CollectableRow) (data.StrategyIntent, error) {
	var (
		intent      data.StrategyIntent
		payloadJSON string
	)
	err := row.Scan(
		&intent.ID,
		&intent.TaskID,
		&intent.TaskType,
		&intent.StrategyID,
		&intent.IntentType,
		&intent.IdempotencyKey,
		&payloadJSON,
		&intent.Policy,
		&intent.Status,
		&intent.CreatedAt,
	)
	if err != nil {
		return data.StrategyIntent{}, err
	}
	payload, err := jsonMap(payloadJSON)
	if err != nil {
		return data.StrategyIntent{}, err
	}
	intent.Payload = payload
	return intent, nil
}

func scanOrder(row pgx.CollectableRow) (data.Order, error) {
	var (
		order                       data.Order
		exchangeResponseSummaryJSON string
	)
	err := row.Scan(
		&order.ID,
		&order.TaskID,
		&order.TaskType,
		&order.IntentID,
		&order.IdempotencyKey,
		&order.Exchange,
		&order.AccountID,
		&order.Symbol,
		&order.Side,
		&order.OrderType,
		&order.Price,
		&order.Quantity,
		&order.Status,
		&order.ExchangeOrderID,
		&exchangeResponseSummaryJSON,
		&order.LastError,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return data.Order{}, err
	}
	summary, err := jsonMap(exchangeResponseSummaryJSON)
	if err != nil {
		return data.Order{}, err
	}
	order.ExchangeResponseSummary = summary
	return order, nil
}

func scanNotification(row pgx.CollectableRow) (data.Notification, error) {
	var notification data.Notification
	err := row.Scan(
		&notification.ID,
		&notification.IntentID,
		&notification.Channel,
		&notification.Title,
		&notification.Body,
		&notification.Status,
		&notification.Error,
		&notification.CreatedAt,
		&notification.SentAt,
	)
	return notification, err
}
