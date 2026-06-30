package datasync

import (
	"context"
	"errors"
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
	if repository.saved.TaskID != "dst_1" || !repository.saved.Completed || len(repository.saved.Candles) != 0 {
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
		heartbeatError:       errors.New("lease lost"),
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
	if repository.failed == nil {
		t.Fatal("expected lease loss to be recorded as failure")
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
	if repository.failed != nil {
		t.Fatalf("shutdown should not mark task failed: %v", repository.failed)
	}
	if repository.saved.TaskID != "" {
		t.Fatalf("shutdown should not save result: %#v", repository.saved)
	}
}

type fakeSyncRepository struct {
	task                  data.DataSyncTask
	claimed               bool
	saved                 data.DataSyncResult
	failed                error
	retry                 error
	retryError            error
	nextAttemptAt         *time.Time
	released              bool
	releasedSkippedFetch  bool
	fetchLockResults      map[string]bool
	fetchLockErr          error
	fetchUnlocks          []string
	fetchLockSkipExchange string
	fetchLockSkippedAt    time.Time
	heartbeats            int
	heartbeatSignals      chan<- struct{}
	heartbeatErrAfter     int
	heartbeatError        error
	heartbeatErrorSignal  chan<- struct{}
}

func (repository *fakeSyncRepository) ClaimDataSyncTask(
	context.Context,
	string,
	time.Duration,
) (data.DataSyncTask, bool, error) {
	return repository.task, repository.claimed, nil
}

func (repository *fakeSyncRepository) HeartbeatDataSyncTask(
	context.Context,
	string,
	string,
	time.Duration,
) error {
	repository.heartbeats++
	if repository.heartbeatSignals != nil {
		select {
		case repository.heartbeatSignals <- struct{}{}:
		default:
		}
	}
	if repository.heartbeatErrAfter > 0 && repository.heartbeats > repository.heartbeatErrAfter {
		if repository.heartbeatErrorSignal != nil {
			select {
			case repository.heartbeatErrorSignal <- struct{}{}:
			default:
			}
		}
		if repository.heartbeatError != nil {
			return repository.heartbeatError
		}
		return errors.New("heartbeat failed")
	}
	return nil
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

func (repository *fakeSyncRepository) RecordDataSyncRetry(
	_ context.Context,
	_ string,
	err error,
	nextAttemptAt *time.Time,
) error {
	repository.retry = err
	repository.nextAttemptAt = nextAttemptAt
	return repository.retryError
}

func (repository *fakeSyncRepository) ReleaseDataSyncTask(context.Context, string) error {
	repository.released = true
	return nil
}

func (repository *fakeSyncRepository) ReleaseDataSyncTaskAfterSkippedFetch(context.Context, string) error {
	repository.releasedSkippedFetch = true
	return nil
}

func (repository *fakeSyncRepository) RecordDataSyncExchangeFetchLockSkipped(
	_ context.Context,
	exchange string,
	skippedAt time.Time,
) error {
	repository.fetchLockSkipExchange = exchange
	repository.fetchLockSkippedAt = skippedAt
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

type blockingMarketClient struct {
	heartbeats <-chan struct{}
	candles    []data.Candle
}

func (client *blockingMarketClient) FetchCandles(
	ctx context.Context,
	_ exchange.CandleRequest,
) ([]data.Candle, error) {
	for index := 0; index < 2; index++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-client.heartbeats:
		}
	}
	return client.candles, nil
}

type leaseLossMarketClient struct {
	leaseLost <-chan struct{}
	candles   []data.Candle
}

func (client *leaseLossMarketClient) FetchCandles(
	context.Context,
	exchange.CandleRequest,
) ([]data.Candle, error) {
	<-client.leaseLost
	return client.candles, nil
}

type cancelingMarketClient struct {
	cancel func()
}

func (client *cancelingMarketClient) FetchCandles(
	ctx context.Context,
	_ exchange.CandleRequest,
) ([]data.Candle, error) {
	client.cancel()
	<-ctx.Done()
	return nil, ctx.Err()
}

func syncTestCandle(openTime time.Time) data.Candle {
	return data.Candle{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		OpenTime:  openTime,
		CloseTime: openTime.Add(time.Minute),
		Open:      "1",
		High:      "2",
		Low:       "1",
		Close:     "2",
		Volume:    "10",
		IsClosed:  true,
	}
}
