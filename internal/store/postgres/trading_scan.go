package postgres

import (
	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

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
		&task.RequestID,
		&task.TraceParent,
		&task.Status,
		&task.LockedBy,
		&task.LockedUntil,
		&task.HeartbeatAt,
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

func scanExecution(row pgx.CollectableRow) (data.Execution, error) {
	var execution data.Execution
	err := row.Scan(
		&execution.ID,
		&execution.TaskID,
		&execution.TaskType,
		&execution.OrderID,
		&execution.IntentID,
		&execution.IdempotencyKey,
		&execution.Exchange,
		&execution.AccountID,
		&execution.Symbol,
		&execution.Side,
		&execution.Price,
		&execution.Quantity,
		&execution.Fee,
		&execution.Status,
		&execution.ExecutedAt,
		&execution.CreatedAt,
	)
	return execution, err
}

func scanPosition(row pgx.CollectableRow) (data.Position, error) {
	var position data.Position
	err := row.Scan(
		&position.TaskID,
		&position.TaskType,
		&position.Exchange,
		&position.AccountID,
		&position.Symbol,
		&position.Quantity,
		&position.AveragePrice,
		&position.RealizedPnL,
		&position.UpdatedAt,
	)
	return position, err
}

func scanNotification(row pgx.CollectableRow) (data.Notification, error) {
	return scanNotificationRow(row)
}

func scanNotificationRow(row rowScanner) (data.Notification, error) {
	var notification data.Notification
	err := row.Scan(
		&notification.ID,
		&notification.TaskID,
		&notification.IntentID,
		&notification.RequestID,
		&notification.TraceParent,
		&notification.Channel,
		&notification.Provider,
		&notification.ProviderMessageID,
		&notification.Target,
		&notification.Title,
		&notification.Body,
		&notification.Status,
		&notification.Error,
		&notification.AttemptCount,
		&notification.MaxAttempts,
		&notification.NextAttemptAt,
		&notification.LastAttemptAt,
		&notification.LastDeliveryDurationMS,
		&notification.CreatedAt,
		&notification.SentAt,
	)
	return notification, err
}
