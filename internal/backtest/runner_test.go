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
	if len(repository.result.Intents) == 0 {
		t.Fatal("expected at least one saved intent")
	}
	intentIDs := map[string]bool{}
	for _, intent := range repository.result.Intents {
		intentIDs[intent.ID] = true
	}
	if !intentIDs[repository.result.Orders[0].IntentID] {
		t.Fatalf("order intent id %s not found in saved intents: %#v", repository.result.Orders[0].IntentID, repository.result.Intents)
	}
	if repository.heartbeats == 0 {
		t.Fatal("expected heartbeat to be refreshed")
	}
	if repository.result.ResultSummary["totalOrders"] == nil {
		t.Fatalf("missing total orders summary: %#v", repository.result.ResultSummary)
	}
}

func TestRunnerRunOnceStoresNotificationIntentsWithoutOrders(t *testing.T) {
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
			StrategyParams: map[string]any{"fastPeriod": 2, "slowPeriod": 3, "orderSize": 0.1, "signalMode": "notification"},
			InitialBalance: "1000",
			Status:         data.TaskStatusPending,
		},
		candles: runnerTestCandles([]string{"10", "9", "8", "11", "12", "10", "8"}),
	}
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(repository.result.Intents) == 0 {
		t.Fatal("expected notification intent to be saved")
	}
	if repository.result.Intents[0].IntentType != strategy.IntentTypeNotification {
		t.Fatalf("intent type = %s", repository.result.Intents[0].IntentType)
	}
	if len(repository.result.Orders) != 0 {
		t.Fatalf("notification intent should not create orders: %#v", repository.result.Orders)
	}
}

func TestRunnerRunOnceUsesOneMinuteExecutionForMinuteReplay(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository := &fakeBacktestRepository{
		task: data.BacktestTask{
			ID:             "bt_1",
			Name:           "EMA",
			Exchange:       "binance",
			Symbol:         "BTCUSDT",
			Interval:       "5m",
			StartTime:      &start,
			EndTime:        ptrTime(start.Add(10 * time.Minute)),
			StrategyID:     "ema-cross",
			StrategyParams: map[string]any{"fastPeriod": 2, "slowPeriod": 3, "orderSize": 0.1, "signalMode": "order"},
			InitialBalance: "1000",
			TriggerMode:    "minute_replay",
			Status:         data.TaskStatusPending,
		},
		candles: runnerTestCandles([]string{"10", "9", "8", "11", "12", "10", "8"}),
	}
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if repository.query.Interval != "1m" {
		t.Fatalf("query interval = %s", repository.query.Interval)
	}
	if repository.result.ResultSummary["executionInterval"] != "1m" {
		t.Fatalf("summary = %#v", repository.result.ResultSummary)
	}
	if repository.result.ResultSummary["triggerMode"] != "minute_replay" {
		t.Fatalf("summary = %#v", repository.result.ResultSummary)
	}
}

func TestRunnerRunOnceIgnoresUnclosedCandleSignals(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := runnerTestCandles([]string{"10", "10", "10", "10", "10", "12"})
	candles[len(candles)-1].IsClosed = false
	repository := &fakeBacktestRepository{
		task: data.BacktestTask{
			ID:             "bt_1",
			Name:           "Breakout",
			Exchange:       "binance",
			Symbol:         "BTCUSDT",
			Interval:       "5m",
			StartTime:      &start,
			EndTime:        ptrTime(start.Add(30 * time.Minute)),
			StrategyID:     "breakout-range",
			StrategyParams: map[string]any{"lookback": 5, "breakoutBufferPct": 0, "orderSize": 0.1, "signalMode": "order", "side": "both"},
			InitialBalance: "1000",
			TriggerMode:    "closed_candle",
			Status:         data.TaskStatusPending,
		},
		candles: candles,
	}
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(repository.result.Intents) != 0 {
		t.Fatalf("unclosed candle should not create intents: %#v", repository.result.Intents)
	}
	if len(repository.result.Orders) != 0 {
		t.Fatalf("unclosed candle should not create orders: %#v", repository.result.Orders)
	}
	if repository.result.ResultSummary["inputCandleCount"] != 6 ||
		repository.result.ResultSummary["strategyCandleCount"] != 5 ||
		repository.result.ResultSummary["droppedOpenCandleCount"] != 1 {
		t.Fatalf("unexpected candle count summary: %#v", repository.result.ResultSummary)
	}
}

func TestRunnerReleasesLeaseOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
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
			Status:         data.TaskStatusRunning,
		},
		cancelOnGetCandles: cancel,
	}
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if !repository.released {
		t.Fatal("expected lease to be released on shutdown")
	}
	if repository.failed {
		t.Fatal("shutdown should not mark backtest failed")
	}
	if repository.saved {
		t.Fatal("shutdown should not save backtest result")
	}
}

type fakeBacktestRepository struct {
	task               data.BacktestTask
	candles            []data.Candle
	query              data.CandleQuery
	result             data.BacktestResult
	claimed            bool
	saved              bool
	failed             bool
	released           bool
	heartbeats         int
	cancelOnGetCandles func()
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

func (repository *fakeBacktestRepository) HeartbeatBacktestTask(
	context.Context,
	string,
	string,
	time.Duration,
) error {
	repository.heartbeats++
	return nil
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

func (repository *fakeBacktestRepository) ReleaseBacktestTask(context.Context, string) error {
	repository.released = true
	return nil
}

func (repository *fakeBacktestRepository) GetCandles(
	ctx context.Context,
	query data.CandleQuery,
) (data.CandleResult, error) {
	repository.query = query
	if repository.cancelOnGetCandles != nil {
		repository.cancelOnGetCandles()
		<-ctx.Done()
		return data.CandleResult{}, ctx.Err()
	}
	return data.CandleResult{
		Candles:           append([]data.Candle(nil), repository.candles...),
		Source:            data.CandleSourceNative,
		RequestedInterval: query.Interval,
		BaseInterval:      query.Interval,
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

func ptrTime(value time.Time) *time.Time {
	return &value
}
