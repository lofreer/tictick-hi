package datasync

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestRunnerUsesRetryAfterForTemporaryFetchError(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:           "dst_1",
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			Interval:     "1m",
			AttemptCount: 1,
		},
		claimed: true,
	}
	fetcher := &fakeMarketClient{
		err: exchange.NewTemporaryErrorWithRetryAfter("temporary 429", nil, 2*time.Minute),
	}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{
		WorkerID:        "test",
		BatchLimit:      10,
		FetchRetries:    3,
		RetryDelay:      time.Nanosecond,
		RetryBackoff:    30 * time.Second,
		MaxRetryBackoff: 5 * time.Minute,
	})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("retry-after fetch should not be retried immediately, calls=%d", fetcher.calls)
	}
	if repository.retry == nil {
		t.Fatal("expected retry to be recorded")
	}
	if repository.nextAttemptAt == nil || !repository.nextAttemptAt.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("nextAttemptAt = %v, want %v", repository.nextAttemptAt, now.Add(2*time.Minute))
	}
	if repository.failed != nil {
		t.Fatalf("temporary error should not mark task failed: %v", repository.failed)
	}
}
