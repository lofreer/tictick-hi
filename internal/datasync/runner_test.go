package datasync

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestRunnerSyncsClaimedTask(t *testing.T) {
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
	fetcher := &fakeMarketClient{
		candles: []data.Candle{{
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
			OpenTime: now.Add(-time.Minute),
			Open:     "1",
			High:     "2",
			Low:      "1",
			Close:    "2",
			Volume:   "10",
			IsClosed: true,
		}},
	}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repository.saved.Candles) != 1 {
		t.Fatalf("saved candles = %d, want 1", len(repository.saved.Candles))
	}
	if repository.saved.LastOpenTime == nil || !repository.saved.Completed {
		t.Fatalf("unexpected result: %#v", repository.saved)
	}
}

func TestRunnerMarksFailedTask(t *testing.T) {
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "missing",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed: true,
	}
	runner := NewRunner(repository, exchange.NewRegistry(nil), Config{WorkerID: "test"})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.failed == nil {
		t.Fatal("expected task failure to be recorded")
	}
}

type fakeSyncRepository struct {
	task    data.DataSyncTask
	claimed bool
	saved   data.DataSyncResult
	failed  error
}

func (repository *fakeSyncRepository) ClaimDataSyncTask(
	context.Context,
	string,
	time.Duration,
) (data.DataSyncTask, bool, error) {
	return repository.task, repository.claimed, nil
}

func (repository *fakeSyncRepository) SaveDataSyncResult(
	_ context.Context,
	result data.DataSyncResult,
) error {
	repository.saved = result
	return nil
}

func (repository *fakeSyncRepository) MarkDataSyncFailed(
	_ context.Context,
	_ string,
	err error,
) error {
	repository.failed = err
	return nil
}

type fakeMarketClient struct {
	candles []data.Candle
	err     error
}

func (client *fakeMarketClient) FetchCandles(
	context.Context,
	exchange.CandleRequest,
) ([]data.Candle, error) {
	if client.err != nil {
		return nil, client.err
	}
	return client.candles, nil
}
