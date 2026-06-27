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
)

type Runner struct {
	repository data.BacktestRepository
	strategies strategy.Repository
	config     Config
}

type Config struct {
	WorkerID     string
	LeaseTTL     time.Duration
	PollInterval time.Duration
	CandleLimit  int
}

func NewRunner(repository data.BacktestRepository, strategies strategy.Repository, config Config) *Runner {
	if config.WorkerID == "" {
		config.WorkerID = "backtest-worker"
	}
	if config.LeaseTTL <= 0 {
		config.LeaseTTL = 30 * time.Second
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
	task, ok, err := runner.repository.ClaimBacktestTask(ctx, runner.config.WorkerID, runner.config.LeaseTTL)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err := runner.runTask(ctx, task); err != nil {
		slog.Error("backtest task failed", "task_id", task.ID, "error", err)
		if markErr := runner.repository.MarkBacktestFailed(ctx, task.ID, err); markErr != nil {
			return fmt.Errorf("mark backtest failed: %w", markErr)
		}
	}
	return nil
}

func (runner *Runner) runTask(ctx context.Context, task data.BacktestTask) error {
	definition, err := runner.strategies.GetStrategy(ctx, task.StrategyID)
	if err != nil {
		return err
	}

	candleResult, err := runner.repository.GetCandles(ctx, data.CandleQuery{
		Exchange: task.Exchange,
		Symbol:   task.Symbol,
		Interval: task.Interval,
		From:     task.StartTime,
		To:       task.EndTime,
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

	orders, summary, err := runner.execute(task, candles, intents)
	if err != nil {
		return err
	}
	return runner.repository.SaveBacktestResult(ctx, data.BacktestResult{
		TaskID:        task.ID,
		Orders:        orders,
		ResultSummary: summary,
	})
}

func (runner *Runner) execute(
	task data.BacktestTask,
	candles []data.Candle,
	intents []strategy.Intent,
) ([]data.BacktestOrder, map[string]any, error) {
	initialBalance, err := decimal.Parse(task.InitialBalance)
	if err != nil {
		return nil, nil, fmt.Errorf("parse initial balance: %w", err)
	}

	cash := initialBalance
	position := decimal.Zero()
	var orders []data.BacktestOrder
	for _, intent := range intents {
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
			IntentID:   intent.ID,
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
