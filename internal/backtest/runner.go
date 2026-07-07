package backtest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/decimal"
	"github.com/lofreer/tictick-hi/internal/strategy"
	"github.com/lofreer/tictick-hi/internal/workerlease"
	"github.com/lofreer/tictick-hi/internal/workerlog"
)

type Runner struct {
	repository data.BacktestRepository
	strategies strategy.Repository
	config     Config
}

type Config struct {
	WorkerID          string
	LeaseTTL          time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	CandleLimit       int
}

func NewRunner(repository data.BacktestRepository, strategies strategy.Repository, config Config) *Runner {
	if config.WorkerID == "" {
		config.WorkerID = "backtest-worker"
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
		config.CandleLimit = 5000
	}
	return &Runner{repository: repository, strategies: strategies, config: config}
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
	task, ok, err := runner.repository.ClaimBacktestTask(ctx, runner.config.WorkerID, runner.config.LeaseTTL)
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
			if releaseErr := runner.repository.ReleaseBacktestTask(releaseCtx, task.ID); releaseErr != nil {
				return fmt.Errorf("release backtest task on shutdown: %w", releaseErr)
			}
			return nil
		}
		slog.Error("backtest task failed", workerlog.TaskAttrs(task.ID, task.RequestID, "error", err)...)
		if markErr := runner.repository.MarkBacktestFailed(ctx, task.ID, err); markErr != nil {
			return fmt.Errorf("mark backtest failed: %w", markErr)
		}
	}
	return nil
}

func (runner *Runner) runTaskWithHeartbeat(ctx context.Context, task data.BacktestTask) (err error) {
	return workerlease.RunWithHeartbeat(
		ctx,
		runner.config.HeartbeatInterval,
		func(heartbeatCtx context.Context) error {
			return runner.repository.HeartbeatBacktestTask(
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

func (runner *Runner) runTask(ctx context.Context, task data.BacktestTask) error {
	definition, err := runner.strategies.GetStrategy(ctx, task.StrategyID)
	if err != nil {
		return err
	}

	triggerMode := backtestTriggerMode(task)
	executionInterval := task.Interval
	if triggerMode == "minute_replay" {
		executionInterval = "1m"
	}
	candleResult, err := runner.repository.GetCandles(ctx, data.CandleQuery{
		Exchange: task.Exchange,
		Symbol:   task.Symbol,
		Interval: executionInterval,
		From:     task.StartTime,
		To:       task.EndTime,
		Limit:    runner.config.CandleLimit,
	})
	if err != nil {
		return err
	}
	if err := data.ValidateStrategyCandleResult(candleResult); err != nil {
		return err
	}
	candles := data.ClosedCandles(candleResult.Candles)

	intents, err := strategy.GenerateIntents(ctx, definition, candles, task.StrategyParams)
	if err != nil {
		return err
	}

	backtestIntents, intentIDs, err := runner.resultIntents(task, intents)
	if err != nil {
		return err
	}
	orders, summary, err := runner.execute(task, candles, intents, intentIDs)
	if err != nil {
		return err
	}
	summary["triggerMode"] = triggerMode
	summary["executionInterval"] = executionInterval
	summary["requestedInterval"] = candleResult.RequestedInterval
	summary["baseInterval"] = candleResult.BaseInterval
	summary["candleSource"] = string(candleResult.Source)
	summary["candleHealth"] = string(candleResult.Health)
	summary["inputCandleCount"] = len(candleResult.Candles)
	summary["strategyCandleCount"] = len(candles)
	summary["droppedOpenCandleCount"] = len(candleResult.Candles) - len(candles)
	return runner.repository.SaveBacktestResult(ctx, data.BacktestResult{
		TaskID:        task.ID,
		Intents:       backtestIntents,
		Orders:        orders,
		ResultSummary: summary,
	})
}

func (runner *Runner) resultIntents(
	task data.BacktestTask,
	intents []strategy.Intent,
) ([]data.StrategyIntent, map[string]string, error) {
	now := time.Now().UTC()
	result := make([]data.StrategyIntent, 0, len(intents))
	intentIDs := make(map[string]string, len(intents))
	for _, intent := range intents {
		intentID, err := core.NewPrefixedID("si")
		if err != nil {
			return nil, nil, err
		}
		intentIDs[intent.ID] = intentID
		result = append(result, data.StrategyIntent{
			ID:             intentID,
			TaskID:         task.ID,
			TaskType:       "backtest",
			StrategyID:     task.StrategyID,
			IntentType:     intent.Type,
			IdempotencyKey: task.ID + ":" + intent.ID,
			Payload:        intentPayload(task, intent),
			Policy:         "simulate",
			Status:         "accepted",
			CreatedAt:      now,
		})
	}
	return result, intentIDs, nil
}

func intentPayload(task data.BacktestTask, intent strategy.Intent) map[string]any {
	payload := map[string]any{}
	for key, value := range intent.Payload {
		payload[key] = value
	}
	payload["taskId"] = task.ID
	payload["taskType"] = "backtest"
	payload["triggerMode"] = backtestTriggerMode(task)
	return payload
}

func backtestTriggerMode(task data.BacktestTask) string {
	if task.TriggerMode == "" {
		return "closed_candle"
	}
	return task.TriggerMode
}

func (runner *Runner) execute(
	task data.BacktestTask,
	candles []data.Candle,
	intents []strategy.Intent,
	intentIDs map[string]string,
) ([]data.BacktestOrder, map[string]any, error) {
	initialBalance, err := decimal.Parse(task.InitialBalance)
	if err != nil {
		return nil, nil, fmt.Errorf("parse initial balance: %w", err)
	}

	cash := initialBalance
	position := decimal.Zero()
	var orders []data.BacktestOrder
	for _, intent := range intents {
		if intent.Type != strategy.IntentTypeOrder {
			continue
		}
		price, err := decimal.Parse(intent.Price)
		if err != nil {
			return nil, nil, fmt.Errorf("parse intent price: %w", err)
		}
		quantity, err := decimal.Parse(intent.Quantity)
		if err != nil {
			return nil, nil, fmt.Errorf("parse intent quantity: %w", err)
		}
		if !quantity.Positive() || !price.Positive() {
			continue
		}

		switch intent.Side {
		case "buy":
			cash = cash.Sub(price.Mul(quantity))
			position = position.Add(quantity)
		case "sell":
			if !position.Positive() {
				continue
			}
			if quantity.GreaterThan(position) {
				quantity = position
			}
			cash = cash.Add(price.Mul(quantity))
			position = position.Sub(quantity)
		default:
			continue
		}

		orderID, err := core.NewPrefixedID("bo")
		if err != nil {
			return nil, nil, err
		}
		orders = append(orders, data.BacktestOrder{
			ID:         orderID,
			BacktestID: task.ID,
			IntentID:   intentIDs[intent.ID],
			Side:       intent.Side,
			Price:      intent.Price,
			Quantity:   quantity.String(),
			Status:     "filled",
			OccurredAt: intent.OccurredAt,
		})
	}

	finalEquity := cash.Add(position.Mul(lastClose(candles)))
	returnPct := decimal.Zero()
	if initialBalance.Positive() {
		returnPct = finalEquity.Sub(initialBalance).Quo(initialBalance).Mul(decimal.NewInt(100))
	}
	return orders, map[string]any{
		"initialBalance": initialBalance.Format(4),
		"finalEquity":    finalEquity.Format(4),
		"returnPct":      returnPct.Format(4),
		"totalIntents":   len(intents),
		"totalOrders":    len(orders),
	}, nil
}

func lastClose(candles []data.Candle) decimal.Decimal {
	if len(candles) == 0 {
		return decimal.Zero()
	}
	price, err := decimal.Parse(candles[len(candles)-1].Close)
	if err != nil {
		return decimal.Zero()
	}
	return price
}
