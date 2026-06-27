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
	if repository.heartbeats == 0 {
		t.Fatal("expected heartbeat to be refreshed")
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
	task                 data.DataSyncTask
	claimed              bool
	saved                data.DataSyncResult
	failed               error
	released             bool
	heartbeats           int
	heartbeatSignals     chan<- struct{}
	heartbeatErrAfter    int
	heartbeatError       error
	heartbeatErrorSignal chan<- struct{}
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

func (repository *fakeSyncRepository) ReleaseDataSyncTask(context.Context, string) error {
	repository.released = true
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
