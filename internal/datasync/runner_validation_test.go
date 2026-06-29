package datasync

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestRunnerRejectsInvalidFetchedCandleBeforeSaving(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:        "dst_1",
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			Status:    data.TaskStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		},
		claimed: true,
	}
	invalidCandle := syncTestCandle(now.Add(-time.Minute))
	invalidCandle.Close = "repair"
	fetcher := &fakeMarketClient{candles: []data.Candle{invalidCandle}}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.saved.TaskID != "" || len(repository.saved.Candles) != 0 {
		t.Fatalf("invalid candle should not be saved: %#v", repository.saved)
	}
	if repository.retry != nil {
		t.Fatalf("invalid candle payload should not be retried as temporary: %v", repository.retry)
	}
	if repository.failed == nil {
		t.Fatal("expected invalid candle payload to fail the task")
	}
	if !strings.Contains(repository.failed.Error(), "validate fetched candles") ||
		!strings.Contains(repository.failed.Error(), "not a decimal") {
		t.Fatalf("failure = %v, want validation error", repository.failed)
	}
}

func TestRunnerRejectsZeroPriceFetchedCandleBeforeSaving(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:        "dst_1",
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			Status:    data.TaskStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		},
		claimed: true,
	}
	invalidCandle := syncTestCandle(now.Add(-time.Minute))
	invalidCandle.Open = "0"
	fetcher := &fakeMarketClient{candles: []data.Candle{invalidCandle}}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.saved.TaskID != "" || len(repository.saved.Candles) != 0 {
		t.Fatalf("zero price candle should not be saved: %#v", repository.saved)
	}
	if repository.retry != nil {
		t.Fatalf("zero price candle should not be retried as temporary: %v", repository.retry)
	}
	if repository.failed == nil {
		t.Fatal("expected zero price candle to fail the task")
	}
	if !strings.Contains(repository.failed.Error(), "validate fetched candles") ||
		!strings.Contains(repository.failed.Error(), "price value must be positive") {
		t.Fatalf("failure = %v, want positive price validation error", repository.failed)
	}
}

func TestRunnerRejectsMismatchedFetchedCandleTargetBeforeSaving(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:        "dst_1",
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			Status:    data.TaskStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		},
		claimed: true,
	}
	mismatchedCandle := syncTestCandle(now.Add(-time.Minute))
	mismatchedCandle.Symbol = "ETHUSDT"
	fetcher := &fakeMarketClient{candles: []data.Candle{mismatchedCandle}}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.saved.TaskID != "" || len(repository.saved.Candles) != 0 {
		t.Fatalf("mismatched candle target should not be saved: %#v", repository.saved)
	}
	if repository.retry != nil {
		t.Fatalf("mismatched candle target should not be retried as temporary: %v", repository.retry)
	}
	if repository.failed == nil {
		t.Fatal("expected mismatched candle target to fail the task")
	}
	if !strings.Contains(repository.failed.Error(), "validate fetched candles") ||
		!strings.Contains(repository.failed.Error(), "target does not match") {
		t.Fatalf("failure = %v, want target mismatch validation error", repository.failed)
	}
}
