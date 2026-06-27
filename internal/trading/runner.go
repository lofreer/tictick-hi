package trading

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
	"github.com/lofreer/tictick-hi/internal/workerlease"
)

type Runner struct {
	repository    data.TradingRepository
	strategies    strategy.Repository
	config        Config
	now           func() time.Time
	paperExecutor OrderExecutor
	liveExecutor  OrderExecutor
}

type Config struct {
	WorkerID          string
	LeaseTTL          time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	CandleLimit       int
}

func NewRunner(repository data.TradingRepository, strategies strategy.Repository, config Config) *Runner {
	if config.WorkerID == "" {
		config.WorkerID = "trading-worker"
	}
	if config.LeaseTTL <= 0 {
		config.LeaseTTL = 30 * time.Second
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = config.LeaseTTL / 3
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = 10 * time.Second
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 10 * time.Second
	}
	if config.CandleLimit <= 0 {
		config.CandleLimit = 500
	}
	return &Runner{
		repository:    repository,
		strategies:    strategies,
		config:        config,
		now:           func() time.Time { return time.Now().UTC() },
		paperExecutor: PaperExecutor{},
		liveExecutor:  LiveExecutor{},
	}
}

func (runner *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(runner.config.PollInterval)
	defer ticker.Stop()

	for {
		if err := runner.RunOnce(ctx); err != nil {
			if workerlease.IsShutdown(ctx, err) {
				return nil
			}
			return err
		}

		select {
		case <-ctx.Done():
			return nil
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

	if err := runner.runTaskWithHeartbeat(ctx, task); err != nil {
		if workerlease.IsShutdown(ctx, err) {
			releaseCtx, cancel := workerlease.ReleaseContext(ctx)
			defer cancel()
			if releaseErr := runner.repository.ReleaseTradingTask(releaseCtx, task.ID); releaseErr != nil {
				return fmt.Errorf("release trading task on shutdown: %w", releaseErr)
			}
			return nil
		}
		slog.Error("trading task failed", "task_id", task.ID, "error", err)
		if markErr := runner.repository.MarkTradingTaskFailed(ctx, task.ID, err); markErr != nil {
			return fmt.Errorf("mark trading task failed: %w", markErr)
		}
	}
	return nil
}

func (runner *Runner) runTaskWithHeartbeat(ctx context.Context, task data.TradingTask) (err error) {
	return workerlease.RunWithHeartbeat(
		ctx,
		runner.config.HeartbeatInterval,
		func(heartbeatCtx context.Context) error {
			return runner.repository.HeartbeatTradingTask(
				heartbeatCtx,
				task.ID,
				runner.config.WorkerID,
				runner.config.LeaseTTL,
			)
		},
		func(runCtx context.Context) error {
			return runner.runTask(runCtx, task)
		},
	)
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

	result, err := runner.result(ctx, task, intents)
	if err != nil {
		return err
	}
	return runner.repository.SaveTradingRunResult(ctx, result)
}

func (runner *Runner) result(
	ctx context.Context,
	task data.TradingTask,
	intents []strategy.Intent,
) (data.TradingRunResult, error) {
	now := runner.now()
	channel := textPolicy(task.IntentPolicy, "notificationChannel", "default")
	result := data.TradingRunResult{TaskID: task.ID}

	for _, intent := range intents {
		idempotencyKey := task.ID + ":" + intent.ID
		intentID := core.StablePrefixedID("si", "intent:"+idempotencyKey)
		policy := intentPolicy(task, intent)

		if policy == "notify" {
			result.Intents = append(result.Intents, tradingIntent(intentRecord{
				Task:           task,
				Intent:         intent,
				IntentID:       intentID,
				IdempotencyKey: idempotencyKey,
				Policy:         policy,
				Status:         "notification_pending",
				Now:            now,
			}))
			notification, err := runner.notification(task, intent, intentID, channel, now)
			if err != nil {
				return data.TradingRunResult{}, err
			}
			result.Notifications = append(result.Notifications, notification)
			continue
		}

		execution, err := runner.executor(task).Execute(ctx, OrderRequest{
			Task:           task,
			Intent:         intent,
			IntentID:       intentID,
			IdempotencyKey: idempotencyKey,
			Now:            now,
		})
		if err != nil {
			return data.TradingRunResult{}, err
		}
		result.Intents = append(result.Intents, tradingIntent(intentRecord{
			Task:           task,
			Intent:         intent,
			IntentID:       intentID,
			IdempotencyKey: idempotencyKey,
			Policy:         policy,
			Status:         "executed",
			Now:            now,
		}))
		result.Orders = append(result.Orders, execution.Order)
		result.Executions = append(result.Executions, execution.Execution)
	}
	return result, nil
}

type intentRecord struct {
	Task           data.TradingTask
	Intent         strategy.Intent
	IntentID       string
	IdempotencyKey string
	Policy         string
	Status         string
	Now            time.Time
}

func tradingIntent(record intentRecord) data.StrategyIntent {
	return data.StrategyIntent{
		ID:             record.IntentID,
		TaskID:         record.Task.ID,
		TaskType:       record.Task.Type,
		StrategyID:     record.Task.StrategyID,
		IntentType:     record.Intent.Type,
		IdempotencyKey: record.IdempotencyKey,
		Payload:        intentPayload(record.Task, record.Intent),
		Policy:         record.Policy,
		Status:         record.Status,
		CreatedAt:      record.Now,
	}
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

func (runner *Runner) notification(
	task data.TradingTask,
	intent strategy.Intent,
	intentID string,
	channel string,
	now time.Time,
) (data.Notification, error) {
	key := "notification:" + task.ID + ":" + intent.ID + ":" + channel
	notificationID := core.StablePrefixedID("nt", key)
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

func intentPolicy(task data.TradingTask, intent strategy.Intent) string {
	if intent.Type == strategy.IntentTypeNotification {
		return "notify"
	}
	if task.Type == "paper" {
		return textPolicy(task.IntentPolicy, "orderIntent", "execute")
	}
	return textPolicy(task.IntentPolicy, "orderIntent", "notify")
}

func (runner *Runner) executor(task data.TradingTask) OrderExecutor {
	if task.Type == "paper" {
		return runner.paperExecutor
	}
	return runner.liveExecutor
}
