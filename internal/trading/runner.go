package trading

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

type Runner struct {
	repository data.TradingRepository
	strategies strategy.Repository
	config     Config
	now        func() time.Time
}

type Config struct {
	WorkerID     string
	LeaseTTL     time.Duration
	PollInterval time.Duration
	CandleLimit  int
}

func NewRunner(repository data.TradingRepository, strategies strategy.Repository, config Config) *Runner {
	if config.WorkerID == "" {
		config.WorkerID = "trading-worker"
	}
	if config.LeaseTTL <= 0 {
		config.LeaseTTL = 30 * time.Second
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 10 * time.Second
	}
	if config.CandleLimit <= 0 {
		config.CandleLimit = 500
	}
	return &Runner{
		repository: repository,
		strategies: strategies,
		config:     config,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (runner *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(runner.config.PollInterval)
	defer ticker.Stop()

	for {
		if err := runner.RunOnce(ctx); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (runner *Runner) RunOnce(ctx context.Context) error {
	task, ok, err := runner.repository.ClaimTradingTask(ctx, runner.config.WorkerID, runner.config.LeaseTTL)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err := runner.runTask(ctx, task); err != nil {
		slog.Error("trading task failed", "task_id", task.ID, "error", err)
		if markErr := runner.repository.MarkTradingTaskFailed(ctx, task.ID, err); markErr != nil {
			return fmt.Errorf("mark trading task failed: %w", markErr)
		}
	}
	return nil
}

func (runner *Runner) runTask(ctx context.Context, task data.TradingTask) error {
	definition, err := runner.strategies.GetStrategy(ctx, task.StrategyID)
	if err != nil {
		return err
	}
	candleResult, err := runner.repository.GetCandles(ctx, data.CandleQuery{
		Exchange: task.Exchange,
		Symbol:   task.Symbol,
		Interval: task.Interval,
		Limit:    runner.config.CandleLimit,
	})
	if err != nil {
		return err
	}
	candles := candleResult.Candles
	intents, err := strategy.GenerateIntents(ctx, definition, candles, task.StrategyParams)
	if err != nil {
		return err
	}

	result, err := runner.result(task, intents)
	if err != nil {
		return err
	}
	return runner.repository.SaveTradingRunResult(ctx, result)
}

func (runner *Runner) result(task data.TradingTask, intents []strategy.Intent) (data.TradingRunResult, error) {
	now := runner.now()
	policy := textPolicy(task.IntentPolicy, "orderIntent", "notify")
	channel := textPolicy(task.IntentPolicy, "notificationChannel", "default")
	result := data.TradingRunResult{TaskID: task.ID}

	for _, intent := range intents {
		idempotencyKey := task.ID + ":" + intent.ID
		intentID, err := core.NewPrefixedID("si")
		if err != nil {
			return data.TradingRunResult{}, err
		}
		result.Intents = append(result.Intents, data.StrategyIntent{
			ID:             intentID,
			TaskID:         task.ID,
			TaskType:       task.Type,
			StrategyID:     task.StrategyID,
			IntentType:     intent.Type,
			IdempotencyKey: idempotencyKey,
			Payload:        intentPayload(task, intent),
			Policy:         policy,
			Status:         "accepted",
			CreatedAt:      now,
		})

		if intent.Type == strategy.IntentTypeNotification {
			notification, err := runner.notification(task, intent, intentID, channel, now)
			if err != nil {
				return data.TradingRunResult{}, err
			}
			result.Notifications = append(result.Notifications, notification)
			continue
		}

		if policy == "execute" {
			order, err := runner.order(task, intent, intentID, idempotencyKey, now)
			if err != nil {
				return data.TradingRunResult{}, err
			}
			result.Orders = append(result.Orders, order)
			continue
		}

		notification, err := runner.notification(task, intent, intentID, channel, now)
		if err != nil {
			return data.TradingRunResult{}, err
		}
		result.Notifications = append(result.Notifications, notification)
	}
	return result, nil
}

func intentPayload(task data.TradingTask, intent strategy.Intent) map[string]any {
	payload := map[string]any{}
	for key, value := range intent.Payload {
		payload[key] = value
	}
	payload["taskId"] = task.ID
	payload["taskType"] = task.Type
	payload["accountId"] = task.AccountID
	return payload
}

func (runner *Runner) order(
	task data.TradingTask,
	intent strategy.Intent,
	intentID string,
	idempotencyKey string,
	now time.Time,
) (data.Order, error) {
	orderID, err := core.NewPrefixedID("ord")
	if err != nil {
		return data.Order{}, err
	}
	status := "filled"
	if task.Type == "live" {
		status = "pending_submission"
	}
	return data.Order{
		ID:                      orderID,
		TaskID:                  task.ID,
		TaskType:                task.Type,
		IntentID:                intentID,
		IdempotencyKey:          idempotencyKey,
		Exchange:                task.Exchange,
		AccountID:               task.AccountID,
		Symbol:                  task.Symbol,
		Side:                    intent.Side,
		OrderType:               "market",
		Price:                   intent.Price,
		Quantity:                intent.Quantity,
		Status:                  status,
		ExchangeResponseSummary: map[string]any{},
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}

func (runner *Runner) notification(
	task data.TradingTask,
	intent strategy.Intent,
	intentID string,
	channel string,
	now time.Time,
) (data.Notification, error) {
	notificationID, err := core.NewPrefixedID("nt")
	if err != nil {
		return data.Notification{}, err
	}
	return data.Notification{
		ID:        notificationID,
		IntentID:  intentID,
		Channel:   channel,
		Title:     "Strategy intent",
		Body:      notificationBody(task, intent),
		Status:    "pending",
		CreatedAt: now,
	}, nil
}

func notificationBody(task data.TradingTask, intent strategy.Intent) string {
	if intent.Message != "" {
		return intent.Message
	}
	return fmt.Sprintf("%s %s %s at %s", intent.Side, intent.Quantity, task.Symbol, intent.Price)
}

func textPolicy(policy map[string]any, key string, fallback string) string {
	value, ok := policy[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return fallback
	}
	return text
}
