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

func TestRunnerRetriesTemporaryFetchError(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed: true,
	}
	fetcher := &fakeMarketClient{
		errs: []error{
			exchange.NewTemporaryError("temporary EOF", nil),
			nil,
		},
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
	}), Config{WorkerID: "test", BatchLimit: 10, FetchRetries: 1, RetryDelay: time.Nanosecond})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 2 {
		t.Fatalf("fetch calls = %d, want 2", fetcher.calls)
	}
	if len(repository.saved.Candles) != 1 {
		t.Fatalf("saved candles = %d, want 1", len(repository.saved.Candles))
	}
	if repository.failed != nil {
		t.Fatalf("unexpected failure: %v", repository.failed)
	}
}

func TestRunnerDoesNotRetryPermanentFetchError(t *testing.T) {
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed: true,
	}
	fetcher := &fakeMarketClient{err: context.Canceled}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10, FetchRetries: 3, RetryDelay: time.Nanosecond})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", fetcher.calls)
	}
	if repository.failed == nil {
		t.Fatal("expected failure")
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
	errs    []error
	calls   int
}

func (client *fakeMarketClient) FetchCandles(
	context.Context,
	exchange.CandleRequest,
) ([]data.Candle, error) {
	client.calls++
	if len(client.errs) > 0 {
		err := client.errs[0]
		client.errs = client.errs[1:]
		if err != nil {
			return nil, err
		}
		return client.candles, nil
	}
	if client.err != nil {
		return nil, client.err
	}
	return client.candles, nil
}
