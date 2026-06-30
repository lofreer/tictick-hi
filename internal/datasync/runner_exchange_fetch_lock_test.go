package datasync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestRunnerReleasesLeaseWithoutFetchWhenExchangeFetchLockHeld(t *testing.T) {
	now := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed:          true,
		fetchLockResults: map[string]bool{"binance": false},
	}
	fetcher := &fakeMarketClient{candles: []data.Candle{syncTestCandle(time.Now().UTC())}}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 0 {
		t.Fatalf("fetch calls = %d, want 0", fetcher.calls)
	}
	if !repository.releasedSkippedFetch || repository.released {
		t.Fatalf(
			"expected skipped fetch release only, releasedSkippedFetch=%v released=%v",
			repository.releasedSkippedFetch,
			repository.released,
		)
	}
	if repository.releasedSkippedWorker != "test" {
		t.Fatalf("skipped fetch released worker id = %q, want test", repository.releasedSkippedWorker)
	}
	if repository.saved.TaskID != "" {
		t.Fatalf("lock-held task should not save result: %#v", repository.saved)
	}
	if repository.failed != nil || repository.retry != nil {
		t.Fatalf("lock-held task should not fail or retry: failed=%v retry=%v", repository.failed, repository.retry)
	}
	if len(repository.fetchUnlocks) != 0 {
		t.Fatalf("fetch unlocks = %#v, want none", repository.fetchUnlocks)
	}
	if repository.fetchLockSkipExchange != "binance" || !repository.fetchLockSkippedAt.Equal(now) {
		t.Fatalf(
			"fetch lock skip = exchange %q at %s, want binance at %s",
			repository.fetchLockSkipExchange,
			repository.fetchLockSkippedAt,
			now,
		)
	}
}

func TestRunnerReturnsInfrastructureErrorWhenExchangeFetchLockFails(t *testing.T) {
	lockErr := errors.New("postgres unavailable")
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed:      true,
		fetchLockErr: lockErr,
	}
	fetcher := &fakeMarketClient{}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})

	err := runner.RunOnce(context.Background())

	var gotLockErr dataSyncExchangeFetchLockError
	if !errors.As(err, &gotLockErr) || !errors.Is(err, lockErr) {
		t.Fatalf("RunOnce error = %v, want dataSyncExchangeFetchLockError wrapping %v", err, lockErr)
	}
	if fetcher.calls != 0 {
		t.Fatalf("fetch calls = %d, want 0", fetcher.calls)
	}
	if !repository.releasedSkippedFetch || repository.released {
		t.Fatalf(
			"expected skipped fetch release only after lock error, releasedSkippedFetch=%v released=%v",
			repository.releasedSkippedFetch,
			repository.released,
		)
	}
	if repository.releasedSkippedWorker != "test" {
		t.Fatalf("skipped fetch released worker id = %q, want test", repository.releasedSkippedWorker)
	}
	if repository.failed != nil || repository.retry != nil {
		t.Fatalf("lock error should not fail or retry task: failed=%v retry=%v", repository.failed, repository.retry)
	}
	if repository.fetchLockSkipExchange != "" {
		t.Fatalf("lock infrastructure error should not record lock-held skip: %q", repository.fetchLockSkipExchange)
	}
}

func (repository *fakeSyncRepository) TryLockDataSyncExchangeFetch(
	_ context.Context,
	exchangeID string,
) (func(context.Context) error, bool, error) {
	if repository.fetchLockErr != nil {
		return nil, false, repository.fetchLockErr
	}
	if locked, ok := repository.fetchLockResults[exchangeID]; ok && !locked {
		return nil, false, nil
	}
	return func(context.Context) error {
		repository.fetchUnlocks = append(repository.fetchUnlocks, exchangeID)
		return nil
	}, true, nil
}
