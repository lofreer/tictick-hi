package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) listTradingIntents(ctx context.Context, taskID string) ([]data.StrategyIntent, error) {
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

func (store *Store) listTradingOrders(ctx context.Context, taskID string) ([]data.Order, error) {
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

func (store *Store) listTradingExecutions(ctx context.Context, taskID string) ([]data.Execution, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, task_id, task_type, order_id, COALESCE(intent_id, ''), idempotency_key,
		       exchange, account_id, symbol, side, price::text, quantity::text, fee::text,
		       status, executed_at, created_at
		  FROM executions
		 WHERE task_id = $1
		 ORDER BY executed_at DESC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list trading executions: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanExecution)
}

func (store *Store) listTradingPositions(ctx context.Context, taskID string) ([]data.Position, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT task_id, task_type, exchange, account_id, symbol, quantity::text,
		       average_price::text, realized_pnl::text, updated_at
		  FROM positions
		 WHERE task_id = $1
		 ORDER BY symbol ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list trading positions: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanPosition)
}

func (store *Store) listTradingNotifications(ctx context.Context, taskID string) ([]data.Notification, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, task_id, COALESCE(intent_id, ''), COALESCE(request_id, ''), channel, provider, target,
		       title, body, status, COALESCE(error, ''), attempt_count,
		       max_attempts, next_attempt_at, last_attempt_at, created_at, sent_at
		  FROM notifications
		 WHERE task_id = $1
		 ORDER BY created_at DESC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list trading notifications: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanNotification)
}

func (store *Store) saveTradingRunResult(ctx context.Context, result data.TradingRunResult) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save trading run result: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, intent := range result.Intents {
		if err := upsertStrategyIntent(ctx, tx, result.TaskID, intent); err != nil {
			return err
		}
	}
	for _, order := range result.Orders {
		if err := upsertOrder(ctx, tx, result.TaskID, order); err != nil {
			return err
		}
	}
	for _, execution := range result.Executions {
		if err := upsertExecution(ctx, tx, result.TaskID, execution); err != nil {
			return err
		}
	}
	for _, notification := range result.Notifications {
		if err := insertNotification(ctx, tx, result.TaskID, notification); err != nil {
			return err
		}
	}
	if err := recalculatePositions(ctx, tx, result.TaskID); err != nil {
		return err
	}
	if err := releaseTradingTask(ctx, tx, result.TaskID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit trading run result: %w", err)
	}
	return nil
}

func upsertStrategyIntent(ctx context.Context, tx pgx.Tx, taskID string, intent data.StrategyIntent) error {
	payloadJSON, err := jsonText(intent.Payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO strategy_intents (
			id, task_id, task_type, strategy_id, intent_type, idempotency_key,
			payload, policy, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)
		ON CONFLICT (task_id, idempotency_key)
		DO UPDATE SET payload = EXCLUDED.payload,
		              policy = EXCLUDED.policy,
		              status = EXCLUDED.status`,
		intent.ID,
		taskID,
		intent.TaskType,
		intent.StrategyID,
		intent.IntentType,
		intent.IdempotencyKey,
		payloadJSON,
		intent.Policy,
		intent.Status,
		intent.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert strategy intent: %w", err)
	}
	return nil
}

func upsertOrder(ctx context.Context, tx pgx.Tx, taskID string, order data.Order) error {
	summaryJSON, err := jsonText(order.ExchangeResponseSummary)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO orders (
			id, task_id, task_type, intent_id, idempotency_key, exchange,
			account_id, symbol, side, order_type, price, quantity, status,
			exchange_order_id, exchange_response_summary, last_error, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		        $11::numeric, $12::numeric, $13, NULLIF($14, ''),
		        $15::jsonb, NULLIF($16, ''), $17, $18)
		ON CONFLICT (task_id, idempotency_key)
		DO UPDATE SET intent_id = EXCLUDED.intent_id,
		              price = EXCLUDED.price,
		              quantity = EXCLUDED.quantity,
		              status = EXCLUDED.status,
		              exchange_response_summary = EXCLUDED.exchange_response_summary,
		              last_error = EXCLUDED.last_error,
		              updated_at = EXCLUDED.updated_at`,
		order.ID,
		taskID,
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
	)
	if err != nil {
		return fmt.Errorf("upsert order: %w", err)
	}
	return nil
}

func upsertExecution(ctx context.Context, tx pgx.Tx, taskID string, execution data.Execution) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO executions (
			id, task_id, task_type, order_id, intent_id, idempotency_key,
			exchange, account_id, symbol, side, price, quantity, fee,
			status, executed_at, created_at
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, $10,
		        $11::numeric, $12::numeric, $13::numeric, $14, $15, $16)
		ON CONFLICT (task_id, idempotency_key)
		DO UPDATE SET order_id = EXCLUDED.order_id,
		              intent_id = EXCLUDED.intent_id,
		              price = EXCLUDED.price,
		              quantity = EXCLUDED.quantity,
		              fee = EXCLUDED.fee,
		              status = EXCLUDED.status,
		              executed_at = EXCLUDED.executed_at`,
		execution.ID,
		taskID,
		execution.TaskType,
		execution.OrderID,
		execution.IntentID,
		execution.IdempotencyKey,
		execution.Exchange,
		execution.AccountID,
		execution.Symbol,
		execution.Side,
		execution.Price,
		execution.Quantity,
		execution.Fee,
		execution.Status,
		execution.ExecutedAt,
		execution.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert execution: %w", err)
	}
	return nil
}

func insertNotification(ctx context.Context, tx pgx.Tx, taskID string, notification data.Notification) error {
	route, err := notificationRoute(ctx, tx, notification.Channel)
	if err != nil {
		return err
	}
	status := notification.Status
	errorText := notification.Error
	if !route.Enabled {
		status = "failed"
		errorText = "notification channel is disabled"
	}
	outboxID := core.StablePrefixedID("no", "outbox:"+notification.ID)
	_, err = tx.Exec(ctx, `
		INSERT INTO notifications (
			id, task_id, intent_id, request_id, channel, provider, target, title, body,
			status, error, attempt_count, max_attempts, next_attempt_at,
			created_at, sent_at, updated_at
		)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8, $9, $10, NULLIF($11, ''),
		        0, 3, $12, $13, $14, $13)
		ON CONFLICT (id) DO NOTHING`,
		notification.ID,
		taskID,
		notification.IntentID,
		notification.RequestID,
		notification.Channel,
		route.Provider,
		route.Target,
		notification.Title,
		notification.Body,
		status,
		errorText,
		notification.CreatedAt,
		notification.CreatedAt,
		notification.SentAt,
	)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO notification_outbox (
			id, notification_id, task_id, intent_id, request_id, channel, provider, target,
			title, body, status, attempt_count, max_attempts, next_attempt_at,
			last_error, created_at, updated_at
		)
		SELECT $1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, $9, $10, $11,
		       0, 3, $12, NULLIF($13, ''), $12, $12
		 WHERE EXISTS (SELECT 1 FROM notifications WHERE id = $2)
		ON CONFLICT (notification_id) DO NOTHING`,
		outboxID,
		notification.ID,
		taskID,
		notification.IntentID,
		notification.RequestID,
		notification.Channel,
		route.Provider,
		route.Target,
		notification.Title,
		notification.Body,
		status,
		notification.CreatedAt,
		errorText,
	)
	if err != nil {
		return fmt.Errorf("insert notification outbox: %w", err)
	}
	return nil
}

type resolvedNotificationRoute struct {
	Provider string
	Target   string
	Enabled  bool
}

func notificationRoute(ctx context.Context, tx pgx.Tx, channel string) (resolvedNotificationRoute, error) {
	route := resolvedNotificationRoute{Provider: "local", Target: channel, Enabled: true}
	row := tx.QueryRow(ctx, `
		SELECT provider, target, enabled
		  FROM notification_channels
		 WHERE name = $1 OR id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`, channel)
	if err := row.Scan(&route.Provider, &route.Target, &route.Enabled); err != nil {
		if err == pgx.ErrNoRows {
			return route, nil
		}
		return resolvedNotificationRoute{}, fmt.Errorf("resolve notification route: %w", err)
	}
	if route.Provider == "" {
		route.Provider = "local"
	}
	if route.Target == "" {
		route.Target = channel
	}
	return route, nil
}

func recalculatePositions(ctx context.Context, tx pgx.Tx, taskID string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM positions WHERE task_id = $1`, taskID); err != nil {
		return fmt.Errorf("clear positions: %w", err)
	}
	_, err := tx.Exec(ctx, `
		WITH grouped AS (
			SELECT task_id, task_type, exchange, account_id, symbol,
			       SUM(CASE WHEN side = 'buy' THEN quantity ELSE -quantity END) AS quantity,
			       SUM(CASE WHEN side = 'buy' THEN quantity ELSE 0 END) AS buy_qty,
			       SUM(CASE WHEN side = 'sell' THEN quantity ELSE 0 END) AS sell_qty,
			       SUM(CASE WHEN side = 'buy' THEN price * quantity ELSE 0 END) AS buy_cost,
			       SUM(CASE WHEN side = 'sell' THEN price * quantity ELSE 0 END) AS sell_proceeds
			  FROM executions
			 WHERE task_id = $1
			 GROUP BY task_id, task_type, exchange, account_id, symbol
		)
		INSERT INTO positions (
			task_id, task_type, exchange, account_id, symbol,
			quantity, average_price, realized_pnl, updated_at
		)
		SELECT task_id, task_type, exchange, account_id, symbol,
		       quantity,
		       CASE
		         WHEN quantity > 0 AND buy_qty > 0 THEN buy_cost / buy_qty
		         WHEN quantity < 0 AND sell_qty > 0 THEN sell_proceeds / sell_qty
		         ELSE 0
		       END,
		       0,
		       now()
		  FROM grouped`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("recalculate positions: %w", err)
	}
	return nil
}

func releaseTradingTask(ctx context.Context, tx pgx.Tx, taskID string) error {
	if err := releaseLease(ctx, tx, tradingTaskLease, taskID); err != nil {
		return fmt.Errorf("release trading task: %w", err)
	}
	return nil
}
