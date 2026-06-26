package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

const tradingTaskColumns = `
	id, name, type, exchange, account_id, symbol, interval, strategy_id,
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

	var id string
	err = tx.QueryRow(ctx, `
		SELECT id
		  FROM trading_tasks
		 WHERE status = $1
		   AND (locked_until IS NULL OR locked_until < now())
		 ORDER BY created_at ASC
		 LIMIT 1
		 FOR UPDATE SKIP LOCKED`,
		data.TaskStatusRunning,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.TradingTask{}, false, nil
	}
	if err != nil {
		return data.TradingTask{}, false, fmt.Errorf("select trading task: %w", err)
	}

	row := tx.QueryRow(ctx, `
		UPDATE trading_tasks
		   SET locked_by = $2,
		       locked_until = now() + $3::interval,
		       heartbeat_at = now(),
		       started_at = COALESCE(started_at, now()),
		       attempt_count = attempt_count + 1,
		       updated_at = now()
		 WHERE id = $1
		RETURNING `+tradingTaskColumns,
		id, workerID, intervalLiteral(leaseTTL),
	)
	task, err := scanTradingTaskRow(row)
	if err != nil {
		return data.TradingTask{}, false, fmt.Errorf("update claimed trading task: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return data.TradingTask{}, false, fmt.Errorf("commit claim trading task: %w", err)
	}
	return task, true, nil
}

func (store *Store) SaveTradingRunResult(ctx context.Context, result data.TradingRunResult) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save trading run result: %w", err)
	}
	defer tx.Rollback(ctx)

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
			return fmt.Errorf("upsert strategy intent: %w", err)
		}
	}

	for _, order := range result.Orders {
		summaryJSON, err := jsonText(order.ExchangeResponseSummary)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO orders (
				id, task_id, task_type, intent_id, idempotency_key, exchange,
				account_id, symbol, side, order_type, price, quantity, status,
				exchange_order_id, exchange_response_summary, last_error, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			        $11::numeric, $12::numeric, $13, NULLIF($14, ''),
			        $15::jsonb, NULLIF($16, ''), $17, $18)
			ON CONFLICT (task_id, idempotency_key)
			DO UPDATE SET status = EXCLUDED.status,
			              exchange_response_summary = EXCLUDED.exchange_response_summary,
			              last_error = EXCLUDED.last_error,
			              updated_at = EXCLUDED.updated_at`,
			order.ID,
			result.TaskID,
			order.TaskType,
			order.IntentID,
			order.IdempotencyKey,
			order.Exchange,
			order.AccountID,
			order.Symbol,
			order.Side,
			order.OrderType,
			order.Price,
			order.Quantity,
			order.Status,
			order.ExchangeOrderID,
			summaryJSON,
			order.LastError,
			order.CreatedAt,
			order.UpdatedAt,
		); err != nil {
			return fmt.Errorf("upsert order: %w", err)
		}
	}

	for _, notification := range result.Notifications {
		if _, err := tx.Exec(ctx, `
			INSERT INTO notifications (
				id, task_id, intent_id, channel, title, body, status, error, created_at, sent_at
			)
			SELECT $1, $2, NULLIF($3, ''), $4, $5, $6, $7, NULLIF($8, ''), $9, $10
			 WHERE NOT EXISTS (
			       SELECT 1 FROM notifications
			        WHERE task_id = $2 AND intent_id = NULLIF($3, '') AND channel = $4
			 )`,
			notification.ID,
			result.TaskID,
			notification.IntentID,
			notification.Channel,
			notification.Title,
			notification.Body,
			notification.Status,
			notification.Error,
			notification.CreatedAt,
			notification.SentAt,
		); err != nil {
			return fmt.Errorf("insert notification: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE trading_tasks
		   SET locked_by = NULL,
		       locked_until = NULL,
		       heartbeat_at = NULL,
		       updated_at = now()
		 WHERE id = $1`,
		result.TaskID,
	); err != nil {
		return fmt.Errorf("release trading task: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit trading run result: %w", err)
	}
	return nil
}

func (store *Store) MarkTradingTaskFailed(ctx context.Context, taskID string, taskErr error) error {
	_, err := store.pool.Exec(ctx, `
		UPDATE trading_tasks
		   SET status = $2,
		       locked_by = NULL,
		       locked_until = NULL,
		       heartbeat_at = NULL,
		       last_error = $3,
		       updated_at = now()
		 WHERE id = $1`,
		taskID,
		data.TaskStatusFailed,
		taskErr.Error(),
	)
	if err != nil {
		return fmt.Errorf("mark trading task failed: %w", err)
	}
	return nil
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
		&task.Interval,
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
