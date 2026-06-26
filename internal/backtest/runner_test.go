package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

func TestRunnerRunOnceSavesOrders(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository := &fakeBacktestRepository{
		task: data.BacktestTask{
			ID:             "bt_1",
			Name:           "EMA",
			Exchange:       "binance",
			Symbol:         "BTCUSDT",
			Interval:       "1m",
			StartTime:      &start,
			EndTime:        ptrTime(start.Add(10 * time.Minute)),
			StrategyID:     "ema-cross",
			StrategyParams: map[string]any{"fastPeriod": 2, "slowPeriod": 3, "orderSize": 0.1, "signalMode": "order"},
			InitialBalance: "1000",
			Status:         data.TaskStatusPending,
		},
		candles: runnerTestCandles([]string{"10", "9", "8", "11", "12", "10", "8"}),
	}
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !repository.saved {
		t.Fatal("expected result to be saved")
	}
	if repository.result.TaskID != "bt_1" {
		t.Fatalf("task id = %s", repository.result.TaskID)
	}
	if len(repository.result.Orders) == 0 {
		t.Fatal("expected at least one saved order")
	}
	if repository.result.ResultSummary["totalOrders"] == nil {
		t.Fatalf("missing total orders summary: %#v", repository.result.ResultSummary)
	}
}

type fakeBacktestRepository struct {
	task    data.BacktestTask
	candles []data.Candle
	result  data.BacktestResult
	claimed bool
	saved   bool
	failed  bool
}

func (repository *fakeBacktestRepository) ClaimBacktestTask(
	context.Context,
	string,
	time.Duration,
) (data.BacktestTask, bool, error) {
	if repository.claimed {
		return data.BacktestTask{}, false, nil
	}
	repository.claimed = true
	return repository.task, true, nil
}

func (repository *fakeBacktestRepository) SaveBacktestResult(
	_ context.Context,
	result data.BacktestResult,
) error {
	repository.saved = true
	repository.result = result
	return nil
}

func (repository *fakeBacktestRepository) MarkBacktestFailed(context.Context, string, error) error {
	repository.failed = true
	return nil
}

func (repository *fakeBacktestRepository) ListCandles(
	context.Context,
	data.CandleQuery,
) ([]data.Candle, error) {
	return append([]data.Candle(nil), repository.candles...), nil
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

func ptrTime(value time.Time) *time.Time {
	return &value
}
