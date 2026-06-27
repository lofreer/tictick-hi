package trading

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

func TestRunnerRunOnceSavesPaperOrder(t *testing.T) {
	repository := newFakeTradingRepository(map[string]any{"orderIntent": "execute"})
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})
	runner.now = func() time.Time { return time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC) }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(repository.result.Intents) == 0 {
		t.Fatal("expected saved strategy intents")
	}
	if len(repository.result.Orders) == 0 {
		t.Fatal("expected saved orders")
	}
	if repository.result.Orders[0].Status != "filled" {
		t.Fatalf("order status = %s", repository.result.Orders[0].Status)
	}
}

func TestRunnerRunOnceSavesNotification(t *testing.T) {
	repository := newFakeTradingRepository(map[string]any{
		"orderIntent":         "notify",
		"notificationChannel": "ops",
	})
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(repository.result.Notifications) == 0 {
		t.Fatal("expected saved notifications")
	}
	if repository.result.Notifications[0].Channel != "ops" {
		t.Fatalf("channel = %s", repository.result.Notifications[0].Channel)
	}
}

type fakeTradingRepository struct {
	task    data.TradingTask
	candles []data.Candle
	result  data.TradingRunResult
	claimed bool
	saved   bool
	failed  bool
}

func newFakeTradingRepository(policy map[string]any) *fakeTradingRepository {
	return &fakeTradingRepository{
		task: data.TradingTask{
			ID:             "tt_1",
			Name:           "Paper EMA",
			Type:           "paper",
			Exchange:       "binance",
			AccountID:      "paper",
			Symbol:         "BTCUSDT",
			Interval:       "1m",
			StrategyID:     "ema-cross",
			StrategyParams: map[string]any{"fastPeriod": 2, "slowPeriod": 3, "orderSize": 0.1, "signalMode": "order"},
			IntentPolicy:   policy,
			Status:         data.TaskStatusRunning,
		},
		candles: runnerTestCandles([]string{"10", "9", "8", "11", "12", "10", "8"}),
	}
}

func (repository *fakeTradingRepository) ClaimTradingTask(
	context.Context,
	string,
	time.Duration,
) (data.TradingTask, bool, error) {
	if repository.claimed {
		return data.TradingTask{}, false, nil
	}
	repository.claimed = true
	return repository.task, true, nil
}

func (repository *fakeTradingRepository) SaveTradingRunResult(
	_ context.Context,
	result data.TradingRunResult,
) error {
	repository.saved = true
	repository.result = result
	return nil
}

func (repository *fakeTradingRepository) MarkTradingTaskFailed(context.Context, string, error) error {
	repository.failed = true
	return nil
}

func (repository *fakeTradingRepository) GetCandles(
	context.Context,
	data.CandleQuery,
) (data.CandleResult, error) {
	return data.CandleResult{
		Candles:           append([]data.Candle(nil), repository.candles...),
		Source:            data.CandleSourceNative,
		RequestedInterval: "1m",
		BaseInterval:      "1m",
		Health:            data.CandleHealthOK,
	}, nil
}

func runnerTestCandles(closes []string) []data.Candle {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]data.Candle, 0, len(closes))
	for index, closePrice := range closes {
		openTime := start.Add(time.Duration(index) * time.Minute)
		candles = append(candles, data.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      closePrice,
			High:      closePrice,
			Low:       closePrice,
			Close:     closePrice,
			Volume:    "1",
			IsClosed:  true,
		})
	}
	return candles
}
