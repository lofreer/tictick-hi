package datasync

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestRunnerIgnoresRetryRecordForDeletedTask(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed:    true,
		retryError: data.ErrNotFound,
	}
	fetcher := &fakeMarketClient{err: exchange.NewTemporaryError("temporary EOF", nil)}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{
		WorkerID:        "test",
		BatchLimit:      10,
		FetchRetries:    1,
		RetryDelay:      time.Nanosecond,
		RetryBackoff:    30 * time.Second,
		MaxRetryBackoff: 5 * time.Minute,
	})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.retry == nil {
		t.Fatal("temporary error should still attempt to record retry")
	}
	if repository.retryWorkerID != "test" {
		t.Fatalf("retry worker id = %q, want test", repository.retryWorkerID)
	}
	if repository.failed != nil {
		t.Fatalf("deleted task retry race should not mark task failed: %v", repository.failed)
	}
	if repository.saved.TaskID != "" {
		t.Fatalf("deleted task retry race should not save result: %#v", repository.saved)
	}
}
