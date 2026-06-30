package datasync

import (
	"context"
	"errors"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

type fakeSyncRepository struct {
	task                  data.DataSyncTask
	claimed               bool
	saved                 data.DataSyncResult
	failed                error
	failedWorkerID        string
	failedError           error
	retry                 error
	retryWorkerID         string
	retryError            error
	nextAttemptAt         *time.Time
	released              bool
	releasedWorkerID      string
	releaseError          error
	releasedSkippedFetch  bool
	releasedSkippedWorker string
	releaseSkippedError   error
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
	workerID string,
	err error,
) error {
	repository.failed = err
	repository.failedWorkerID = workerID
	return repository.failedError
}

func (repository *fakeSyncRepository) RecordDataSyncRetry(
	_ context.Context,
	_ string,
	workerID string,
	err error,
	nextAttemptAt *time.Time,
) error {
	repository.retry = err
	repository.retryWorkerID = workerID
	repository.nextAttemptAt = nextAttemptAt
	return repository.retryError
}

func (repository *fakeSyncRepository) ReleaseDataSyncTask(_ context.Context, _ string, workerID string) error {
	repository.released = true
	repository.releasedWorkerID = workerID
	return repository.releaseError
}

func (repository *fakeSyncRepository) ReleaseDataSyncTaskAfterSkippedFetch(
	_ context.Context,
	_ string,
	workerID string,
) error {
	repository.releasedSkippedFetch = true
	repository.releasedSkippedWorker = workerID
	return repository.releaseSkippedError
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
