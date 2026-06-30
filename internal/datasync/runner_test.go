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
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  now.Add(-time.Minute),
			CloseTime: now,
			Open:      "1",
			High:      "2",
			Low:       "1",
			Close:     "2",
			Volume:    "10",
			IsClosed:  true,
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
	if repository.saved.WorkerID != "test" {
		t.Fatalf("saved worker id = %q, want test", repository.saved.WorkerID)
	}
	if repository.saved.LastOpenTime == nil || !repository.saved.Completed {
		t.Fatalf("unexpected result: %#v", repository.saved)
	}
	if repository.heartbeats == 0 {
		t.Fatal("expected heartbeat to be refreshed")
	}
}

func TestRunnerCompletesOneShotTaskAlreadySyncedThroughEndWithoutExchangeClient(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 1, 1, 1, 59, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:                   "dst_1",
			Exchange:             "binance",
			Symbol:               "S8SEEDEDUSDT",
			Interval:             "1m",
			StartTime:            &start,
			EndTime:              &end,
			SyncEnabled:          true,
			RealtimeEnabled:      false,
			Status:               data.TaskStatusRunning,
			LatestSyncedOpenTime: &latest,
		},
		claimed: true,
	}
	runner := NewRunner(repository, exchange.NewRegistry(nil), Config{WorkerID: "test", BatchLimit: 10, OverlapCandles: 2})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.saved.TaskID != "dst_1" || repository.saved.WorkerID != "test" ||
		!repository.saved.Completed || len(repository.saved.Candles) != 0 {
		t.Fatalf("unexpected saved result: %#v", repository.saved)
	}
	if repository.failed != nil || repository.retry != nil {
		t.Fatalf("task should complete without failure or retry, failed=%v retry=%v", repository.failed, repository.retry)
	}
}

func TestRunnerDoesNotAdvanceCursorPastFetchedGap(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Minute)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:          "dst_1",
			Exchange:    "binance",
			Symbol:      "BTCUSDT",
			Interval:    "1m",
			StartTime:   &start,
			EndTime:     &end,
			SyncEnabled: true,
			Status:      data.TaskStatusRunning,
		},
		claimed: true,
	}
	fetcher := &fakeMarketClient{
		candles: []data.Candle{
			syncTestCandle(start),
			syncTestCandle(start.Add(time.Minute)),
			syncTestCandle(start.Add(3 * time.Minute)),
			syncTestCandle(start.Add(4 * time.Minute)),
		},
	}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})
	runner.now = func() time.Time { return end }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantCursor := start.Add(time.Minute)
	if repository.saved.LastOpenTime == nil || !repository.saved.LastOpenTime.Equal(wantCursor) {
		t.Fatalf("cursor = %v, want %v", repository.saved.LastOpenTime, wantCursor)
	}
	if repository.saved.WorkerID != "test" {
		t.Fatalf("saved worker id = %q, want test", repository.saved.WorkerID)
	}
	if repository.saved.Completed {
		t.Fatalf("gapped batch should not complete task: %#v", repository.saved)
	}
}

func TestRunnerDoesNotAdvanceCursorWhenOverlapGapRemains(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(6 * time.Minute)
	latest := start.Add(3 * time.Minute)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:                   "dst_1",
			Exchange:             "binance",
			Symbol:               "BTCUSDT",
			Interval:             "1m",
			StartTime:            &start,
			EndTime:              &end,
			SyncEnabled:          true,
			Status:               data.TaskStatusRunning,
			LatestSyncedOpenTime: &latest,
		},
		claimed: true,
	}
	fetcher := &fakeMarketClient{
		candles: []data.Candle{
			syncTestCandle(start.Add(time.Minute)),
			syncTestCandle(start.Add(3 * time.Minute)),
			syncTestCandle(start.Add(4 * time.Minute)),
			syncTestCandle(start.Add(5 * time.Minute)),
		},
	}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10, OverlapCandles: 2})
	runner.now = func() time.Time { return end }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.saved.LastOpenTime != nil {
		t.Fatalf("cursor should not advance across overlap gap: %#v", repository.saved)
	}
	if repository.saved.Completed {
		t.Fatalf("overlap gap should not complete task: %#v", repository.saved)
	}
}

func TestRunnerAdvancesCursorAfterOverlapGapIsFilled(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(6 * time.Minute)
	latest := start.Add(3 * time.Minute)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:                   "dst_1",
			Exchange:             "binance",
			Symbol:               "BTCUSDT",
			Interval:             "1m",
			StartTime:            &start,
			EndTime:              &end,
			SyncEnabled:          true,
			Status:               data.TaskStatusRunning,
			LatestSyncedOpenTime: &latest,
		},
		claimed: true,
	}
	fetcher := &fakeMarketClient{
		candles: []data.Candle{
			syncTestCandle(start.Add(time.Minute)),
			syncTestCandle(start.Add(2 * time.Minute)),
			syncTestCandle(start.Add(3 * time.Minute)),
			syncTestCandle(start.Add(4 * time.Minute)),
			syncTestCandle(start.Add(5 * time.Minute)),
		},
	}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10, OverlapCandles: 2})
	runner.now = func() time.Time { return end }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantCursor := start.Add(5 * time.Minute)
	if repository.saved.LastOpenTime == nil || !repository.saved.LastOpenTime.Equal(wantCursor) {
		t.Fatalf("cursor = %v, want %v", repository.saved.LastOpenTime, wantCursor)
	}
	if !repository.saved.Completed {
		t.Fatalf("filled overlap should complete through end: %#v", repository.saved)
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
	if repository.failedWorkerID != "test" {
		t.Fatalf("failed worker id = %q, want test", repository.failedWorkerID)
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
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  now.Add(-time.Minute),
			CloseTime: now,
			Open:      "1",
			High:      "2",
			Low:       "1",
			Close:     "2",
			Volume:    "10",
			IsClosed:  true,
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

func TestRunnerRecordsTemporaryFetchErrorForRetry(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:           "dst_1",
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			Interval:     "1m",
			AttemptCount: 2,
		},
		claimed: true,
	}
	fetcher := &fakeMarketClient{
		err: exchange.NewTemporaryError("temporary EOF", nil),
	}
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
	if fetcher.calls != 2 {
		t.Fatalf("fetch calls = %d, want 2", fetcher.calls)
	}
	if repository.retry == nil {
		t.Fatal("expected retry to be recorded")
	}
	if repository.retryWorkerID != "test" {
		t.Fatalf("retry worker id = %q, want test", repository.retryWorkerID)
	}
	if repository.nextAttemptAt == nil || !repository.nextAttemptAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("nextAttemptAt = %v, want %v", repository.nextAttemptAt, now.Add(time.Minute))
	}
	if repository.failed != nil {
		t.Fatalf("temporary error should not mark task failed: %v", repository.failed)
	}
}

func TestBoundedExponentialBackoffCapsDelay(t *testing.T) {
	base := 30 * time.Second
	maxDelay := 2 * time.Minute

	if delay := boundedExponentialBackoff(base, maxDelay, 1); delay != 30*time.Second {
		t.Fatalf("attempt 1 delay = %s", delay)
	}
	if delay := boundedExponentialBackoff(base, maxDelay, 3); delay != 2*time.Minute {
		t.Fatalf("attempt 3 delay = %s", delay)
	}
	if delay := boundedExponentialBackoff(base, maxDelay, 8); delay != 2*time.Minute {
		t.Fatalf("attempt 8 delay = %s", delay)
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
	if repository.failedWorkerID != "test" {
		t.Fatalf("failed worker id = %q, want test", repository.failedWorkerID)
	}
}

func TestRunnerRefreshesHeartbeatWhileFetching(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	heartbeats := make(chan struct{}, 4)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed:          true,
		heartbeatSignals: heartbeats,
	}
	fetcher := &blockingMarketClient{
		heartbeats: heartbeats,
		candles: []data.Candle{{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  now.Add(-time.Minute),
			CloseTime: now,
			Open:      "1",
			High:      "2",
			Low:       "1",
			Close:     "2",
			Volume:    "10",
			IsClosed:  true,
		}},
	}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10, LeaseTTL: time.Second, HeartbeatInterval: time.Millisecond})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.heartbeats < 3 {
		t.Fatalf("heartbeats = %d, want at least 3", repository.heartbeats)
	}
	if len(repository.saved.Candles) != 1 {
		t.Fatalf("saved candles = %d, want 1", len(repository.saved.Candles))
	}
}

func TestRunnerDoesNotSaveAfterHeartbeatLeaseLoss(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	leaseLost := make(chan struct{}, 1)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed:              true,
		heartbeatErrAfter:    1,
		heartbeatError:       data.DataSyncCommandInvalidStateError(),
		heartbeatErrorSignal: leaseLost,
	}
	fetcher := &leaseLossMarketClient{
		leaseLost: leaseLost,
		candles: []data.Candle{{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  now.Add(-time.Minute),
			CloseTime: now,
			Open:      "1",
			High:      "2",
			Low:       "1",
			Close:     "2",
			Volume:    "10",
			IsClosed:  true,
		}},
	}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10, LeaseTTL: time.Second, HeartbeatInterval: time.Millisecond})
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.saved.TaskID != "" {
		t.Fatalf("result should not be saved after lease loss: %#v", repository.saved)
	}
	if repository.failed != nil {
		t.Fatalf("lease loss should not be recorded as task failure: %v", repository.failed)
	}
}

func TestRunnerIgnoresRetryRecordLeaseRace(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 10, 0, 0, time.UTC)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed:    true,
		retryError: data.DataSyncCommandInvalidStateError(),
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
	if repository.failed != nil {
		t.Fatalf("retry ownership race should not mark task failed: %v", repository.failed)
	}
	if repository.saved.TaskID != "" {
		t.Fatalf("retry ownership race should not save result: %#v", repository.saved)
	}
}

func TestRunnerIgnoresFailureRecordLeaseRace(t *testing.T) {
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed:     true,
		failedError: data.DataSyncCommandInvalidStateError(),
	}
	fetcher := &fakeMarketClient{err: context.Canceled}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10, FetchRetries: 3, RetryDelay: time.Nanosecond})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.failed == nil {
		t.Fatal("permanent error should still attempt to record failure")
	}
	if repository.failedWorkerID != "test" {
		t.Fatalf("failed worker id = %q, want test", repository.failedWorkerID)
	}
	if repository.saved.TaskID != "" {
		t.Fatalf("failure ownership race should not save result: %#v", repository.saved)
	}
}

func TestRunnerReleasesLeaseOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:       "dst_1",
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			Interval: "1m",
		},
		claimed: true,
	}
	fetcher := &cancelingMarketClient{cancel: cancel}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if !repository.released {
		t.Fatal("expected lease to be released on shutdown")
	}
	if repository.releasedWorkerID != "test" {
		t.Fatalf("released worker id = %q, want test", repository.releasedWorkerID)
	}
	if repository.failed != nil {
		t.Fatalf("shutdown should not mark task failed: %v", repository.failed)
	}
	if repository.saved.TaskID != "" {
		t.Fatalf("shutdown should not save result: %#v", repository.saved)
	}
}
