package trading

import (
	"context"
	"strings"
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
	if len(repository.result.Executions) == 0 {
		t.Fatal("expected saved executions")
	}
	if repository.result.Executions[0].OrderID != repository.result.Orders[0].ID {
		t.Fatalf("execution order id = %s, order id = %s", repository.result.Executions[0].OrderID, repository.result.Orders[0].ID)
	}
	if repository.result.Orders[0].Status != "filled" {
		t.Fatalf("order status = %s", repository.result.Orders[0].Status)
	}
	if repository.result.Intents[0].Status != "executed" {
		t.Fatalf("intent status = %s", repository.result.Intents[0].Status)
	}
	if repository.heartbeats == 0 {
		t.Fatal("expected heartbeat to be refreshed")
	}
}

func TestRunnerRunOnceSavesNotification(t *testing.T) {
	repository := newFakeTradingRepository(map[string]any{
		"orderIntent":         "notify",
		"notificationChannel": "ops",
	})
	repository.task.RequestID = "request-id-notification"
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
	if repository.result.Notifications[0].RequestID != "request-id-notification" {
		t.Fatalf("notification request id = %q", repository.result.Notifications[0].RequestID)
	}
}

func TestRunnerRunOncePersistsStrategyNotificationIntent(t *testing.T) {
	repository := newFakeTradingRepository(map[string]any{"orderIntent": "execute"})
	repository.task.StrategyParams["signalMode"] = "notification"
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(repository.result.Intents) == 0 {
		t.Fatal("expected saved strategy intent")
	}
	if repository.result.Intents[0].IntentType != strategy.IntentTypeNotification {
		t.Fatalf("intent type = %s", repository.result.Intents[0].IntentType)
	}
	if len(repository.result.Orders) != 0 {
		t.Fatalf("notification intent should not create orders: %#v", repository.result.Orders)
	}
	if len(repository.result.Executions) != 0 {
		t.Fatalf("notification intent should not create executions: %#v", repository.result.Executions)
	}
	if len(repository.result.Notifications) == 0 {
		t.Fatal("expected notification record")
	}
}

func TestRunnerRunOnceIgnoresUnclosedCandleSignals(t *testing.T) {
	repository := newFakeTradingRepository(map[string]any{"orderIntent": "execute"})
	repository.task.Name = "Paper Breakout"
	repository.task.Interval = "5m"
	repository.task.StrategyID = "breakout-range"
	repository.task.StrategyParams = map[string]any{
		"lookback":          5,
		"breakoutBufferPct": 0,
		"orderSize":         0.1,
		"signalMode":        "order",
		"side":              "both",
	}
	repository.candles = runnerTestCandles([]string{"10", "10", "10", "10", "10", "12"})
	repository.candles[len(repository.candles)-1].IsClosed = false
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !repository.saved {
		t.Fatal("expected empty trading run result to be saved")
	}
	if len(repository.result.Intents) != 0 {
		t.Fatalf("unclosed candle should not create intents: %#v", repository.result.Intents)
	}
	if len(repository.result.Orders) != 0 || len(repository.result.Executions) != 0 {
		t.Fatalf("unclosed candle should not create orders or executions: %#v %#v", repository.result.Orders, repository.result.Executions)
	}
}

func TestRunnerRunOnceFailsOnLimitedCoverage(t *testing.T) {
	repository := newFakeTradingRepository(map[string]any{"orderIntent": "execute"})
	repository.candleResult = &data.CandleResult{
		Candles:           runnerTestCandles([]string{"10", "9", "8", "11", "12", "10", "8"}),
		Source:            data.CandleSourceAggregated,
		RequestedInterval: "1h",
		BaseInterval:      "1m",
		Health:            data.CandleHealthOK,
		Coverage: data.CandleCoverage{
			RequestedLimit:      1000,
			ReturnedCandles:     85,
			RequiredBaseCandles: 60000,
			BaseLimit:           data.MaxCandleLimit,
			ReturnedBaseCandles: data.MaxCandleLimit,
			LimitedByBaseWindow: true,
		},
	}
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !repository.failed {
		t.Fatal("expected trading task to be marked failed")
	}
	if repository.saved {
		t.Fatal("limited candle coverage should not save trading result")
	}
	if repository.failedErr == nil || !strings.Contains(repository.failedErr.Error(), "candle data coverage is limited") {
		t.Fatalf("unexpected failure error: %v", repository.failedErr)
	}
}

func TestRunnerRunOnceRejectsLiveExecute(t *testing.T) {
	repository := newFakeTradingRepository(map[string]any{"orderIntent": "execute"})
	repository.task.Type = "live"
	repository.task.AccountID = "acct_live"
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	if repository.saved {
		t.Fatal("live execution should not save a fake local order")
	}
	if !repository.failed {
		t.Fatal("expected live execution task to be marked failed")
	}
}

func TestRunnerReleasesLeaseOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repository := newFakeTradingRepository(map[string]any{"orderIntent": "execute"})
	repository.cancelOnGetCandles = cancel
	runner := NewRunner(repository, strategy.BuiltinRegistry(), Config{WorkerID: "test-worker"})

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if !repository.released {
		t.Fatal("expected lease to be released on shutdown")
	}
	if repository.failed {
		t.Fatal("shutdown should not mark trading task failed")
	}
	if repository.saved {
		t.Fatal("shutdown should not save trading result")
	}
}

type fakeTradingRepository struct {
	task               data.TradingTask
	candles            []data.Candle
	candleResult       *data.CandleResult
	result             data.TradingRunResult
	claimed            bool
	saved              bool
	failed             bool
	failedErr          error
	released           bool
	heartbeats         int
	cancelOnGetCandles func()
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

func (repository *fakeTradingRepository) HeartbeatTradingTask(
	context.Context,
	string,
	string,
	time.Duration,
) error {
	repository.heartbeats++
	return nil
}

func (repository *fakeTradingRepository) SaveTradingRunResult(
	_ context.Context,
	result data.TradingRunResult,
) error {
	repository.saved = true
	repository.result = result
	return nil
}

func (repository *fakeTradingRepository) MarkTradingTaskFailed(_ context.Context, _ string, err error) error {
	repository.failed = true
	repository.failedErr = err
	return nil
}

func (repository *fakeTradingRepository) ReleaseTradingTask(context.Context, string) error {
	repository.released = true
	return nil
}

func (repository *fakeTradingRepository) GetCandles(
	ctx context.Context,
	_ data.CandleQuery,
) (data.CandleResult, error) {
	if repository.cancelOnGetCandles != nil {
		repository.cancelOnGetCandles()
		<-ctx.Done()
		return data.CandleResult{}, ctx.Err()
	}
	if repository.candleResult != nil {
		result := *repository.candleResult
		result.Candles = append([]data.Candle(nil), repository.candleResult.Candles...)
		return result, nil
	}
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
