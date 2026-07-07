package datasync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
	"github.com/lofreer/tictick-hi/internal/workerlease"
	"github.com/lofreer/tictick-hi/internal/workerlog"
)

type Runner struct {
	repository     data.SyncRepository
	lockRepository data.SyncFetchLockRepository
	registry       exchange.Registry
	config         Config
	now            func() time.Time
}

type Config struct {
	WorkerID          string
	LeaseTTL          time.Duration
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	BatchLimit        int
	OverlapCandles    int
	DefaultLookback   time.Duration
	FetchRetries      int
	RetryDelay        time.Duration
	RetryBackoff      time.Duration
	MaxRetryBackoff   time.Duration
}

const RealtimeSyncModeRESTPolling = "rest_polling"

var errDataSyncExchangeFetchLockHeld = errors.New("data sync exchange fetch lock is held")

type dataSyncExchangeFetchLockError struct {
	err error
}

func (lockErr dataSyncExchangeFetchLockError) Error() string {
	return fmt.Sprintf("data sync exchange fetch lock: %v", lockErr.err)
}

func (lockErr dataSyncExchangeFetchLockError) Unwrap() error {
	return lockErr.err
}

func NewRunner(repository data.SyncRepository, registry exchange.Registry, config Config) *Runner {
	if config.LeaseTTL <= 0 {
		config.LeaseTTL = 30 * time.Second
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = config.LeaseTTL / 3
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = 10 * time.Second
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 10 * time.Second
	}
	if config.BatchLimit <= 0 {
		config.BatchLimit = 500
	}
	if config.OverlapCandles < 0 {
		config.OverlapCandles = 0
	}
	if config.DefaultLookback <= 0 {
		config.DefaultLookback = 500 * time.Minute
	}
	if config.FetchRetries <= 0 {
		config.FetchRetries = 2
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 250 * time.Millisecond
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = 30 * time.Second
	}
	if config.MaxRetryBackoff <= 0 {
		config.MaxRetryBackoff = 5 * time.Minute
	}
	if config.MaxRetryBackoff < config.RetryBackoff {
		config.MaxRetryBackoff = config.RetryBackoff
	}
	if config.WorkerID == "" {
		config.WorkerID = "sync-worker"
	}
	lockRepository, _ := repository.(data.SyncFetchLockRepository)

	return &Runner{
		repository:     repository,
		lockRepository: lockRepository,
		registry:       registry,
		config:         config,
		now:            func() time.Time { return time.Now().UTC() },
	}
}

func (runner *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(runner.config.PollInterval)
	defer ticker.Stop()

	for {
		if err := runner.RunOnce(ctx); err != nil {
			if workerlease.IsShutdown(ctx, err) {
				return nil
			}
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (runner *Runner) RunOnce(ctx context.Context) error {
	task, ok, err := runner.repository.ClaimDataSyncTask(ctx, runner.config.WorkerID, runner.config.LeaseTTL)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err := runner.syncTaskWithHeartbeat(ctx, task); err != nil {
		if workerlease.IsShutdown(ctx, err) {
			releaseCtx, cancel := workerlease.ReleaseContext(ctx)
			defer cancel()
			if releaseErr := runner.releaseDataSyncTaskOnShutdown(releaseCtx, task); releaseErr != nil {
				return releaseErr
			}
			return nil
		}
		if errors.Is(err, errDataSyncExchangeFetchLockHeld) {
			releaseCtx, cancel := workerlease.ReleaseContext(ctx)
			defer cancel()
			if recordErr := runner.recordDataSyncExchangeFetchLockSkipped(releaseCtx, task.Exchange); recordErr != nil {
				slog.Error(
					"record data sync exchange fetch lock skip failed",
					workerlog.TaskTraceAttrs(
						task.ID,
						task.RequestID,
						task.TraceParent,
						"exchange",
						task.Exchange,
						"error",
						recordErr,
					)...,
				)
			}
			if releaseErr := runner.releaseDataSyncTaskAfterExchangeFetchLockSkip(releaseCtx, task); releaseErr != nil {
				return releaseErr
			}
			slog.Info(
				"data sync task released; exchange fetch lock held",
				workerlog.TaskTraceAttrs(task.ID, task.RequestID, task.TraceParent, "exchange", task.Exchange)...,
			)
			return nil
		}
		var lockErr dataSyncExchangeFetchLockError
		if errors.As(err, &lockErr) {
			releaseCtx, cancel := workerlease.ReleaseContext(ctx)
			defer cancel()
			if releaseErr := runner.releaseDataSyncTaskAfterExchangeFetchLockError(releaseCtx, task); releaseErr != nil {
				return releaseErr
			}
			return lockErr
		}
		if isDataSyncLeaseRace(err) {
			slog.Info(
				"data sync task no longer owned by worker",
				workerlog.TaskTraceAttrs(task.ID, task.RequestID, task.TraceParent, "error", err)...,
			)
			return nil
		}
		if exchange.IsTemporaryError(err) {
			nextAttemptAt := runner.nextAttemptAt(task, err)
			slog.Warn(
				"data sync task will retry after temporary market data error",
				workerlog.TaskTraceAttrs(
					task.ID,
					task.RequestID,
					task.TraceParent,
					"next_attempt_at",
					nextAttemptAt,
					"error",
					err,
				)...,
			)
			if retryErr := runner.repository.RecordDataSyncRetry(ctx, task.ID, runner.config.WorkerID, err, &nextAttemptAt); retryErr != nil {
				if isDataSyncLeaseRace(retryErr) {
					slog.Info(
						"data sync task no longer owned before retry was recorded",
						workerlog.TaskTraceAttrs(task.ID, task.RequestID, task.TraceParent, "error", retryErr)...,
					)
					return nil
				}
				return fmt.Errorf("record data sync retry: %w", retryErr)
			}
			return nil
		}
		slog.Error(
			"data sync task failed",
			workerlog.TaskTraceAttrs(task.ID, task.RequestID, task.TraceParent, "error", err)...,
		)
		if markErr := runner.repository.MarkDataSyncFailed(ctx, task.ID, runner.config.WorkerID, err); markErr != nil {
			if isDataSyncLeaseRace(markErr) {
				slog.Info(
					"data sync task no longer owned before failure was recorded",
					workerlog.TaskTraceAttrs(task.ID, task.RequestID, task.TraceParent, "error", markErr)...,
				)
				return nil
			}
			return fmt.Errorf("mark data sync failed: %w", markErr)
		}
	}
	return nil
}

func (runner *Runner) recordDataSyncExchangeFetchLockSkipped(ctx context.Context, exchange string) error {
	if runner.lockRepository == nil {
		return nil
	}
	return runner.lockRepository.RecordDataSyncExchangeFetchLockSkipped(ctx, exchange, runner.now())
}

func (runner *Runner) syncTaskWithHeartbeat(ctx context.Context, task data.DataSyncTask) (err error) {
	return workerlease.RunWithHeartbeat(
		ctx,
		runner.config.HeartbeatInterval,
		func(heartbeatCtx context.Context) error {
			return runner.repository.HeartbeatDataSyncTask(
				heartbeatCtx,
				task.ID,
				runner.config.WorkerID,
				runner.config.LeaseTTL,
			)
		},
		func(runCtx context.Context) error {
			return runner.syncTask(runCtx, task)
		},
	)
}

func (runner *Runner) syncTask(ctx context.Context, task data.DataSyncTask) error {
	duration, err := data.IntervalDuration(task.Interval)
	if err != nil {
		return err
	}

	if isAlreadySyncedThroughEnd(task, duration) {
		return runner.repository.SaveDataSyncResult(ctx, data.DataSyncResult{
			TaskID:    task.ID,
			WorkerID:  runner.config.WorkerID,
			Completed: true,
		})
	}

	client, err := runner.registry.Client(task.Exchange)
	if err != nil {
		return err
	}

	window := runner.syncWindow(task, duration)
	if !window.from.Before(window.to) {
		return runner.repository.SaveDataSyncResult(ctx, data.DataSyncResult{
			TaskID:    task.ID,
			WorkerID:  runner.config.WorkerID,
			Completed: !task.RealtimeEnabled,
		})
	}

	if runner.lockRepository == nil {
		return dataSyncExchangeFetchLockError{err: errors.New("sync repository does not support exchange fetch locks")}
	}
	unlock, locked, err := runner.lockRepository.TryLockDataSyncExchangeFetch(ctx, task.Exchange)
	if err != nil {
		return dataSyncExchangeFetchLockError{err: err}
	}
	if !locked {
		return errDataSyncExchangeFetchLockHeld
	}

	candles, err := runner.fetchCandles(ctx, task, client, exchange.CandleRequest{
		Exchange: task.Exchange,
		Symbol:   task.Symbol,
		Interval: task.Interval,
		From:     window.from,
		To:       window.to,
		Limit:    runner.config.BatchLimit,
	})
	if unlockErr := unlockDataSyncExchangeFetch(unlock); unlockErr != nil {
		return dataSyncExchangeFetchLockError{err: unlockErr}
	}
	if err != nil {
		return err
	}
	closedCandles := data.ClosedCandles(candles)
	if err := data.ValidateCandleSeriesForTarget(closedCandles, task.Exchange, task.Symbol, task.Interval); err != nil {
		return fmt.Errorf("validate fetched candles: %w", err)
	}

	cursorOpenTime := nextCursorOpenTime(task, duration, closedCandles)
	if err := runner.repository.HeartbeatDataSyncTask(ctx, task.ID, runner.config.WorkerID, runner.config.LeaseTTL); err != nil {
		return err
	}
	return runner.repository.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:       task.ID,
		WorkerID:     runner.config.WorkerID,
		Candles:      closedCandles,
		LastOpenTime: cursorOpenTime,
		Completed:    runner.isCompleted(task, duration, cursorOpenTime, len(candles) == 0, len(closedCandles) > 0),
	})
}

func unlockDataSyncExchangeFetch(unlock func(context.Context) error) error {
	releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return unlock(releaseCtx)
}

func (runner *Runner) nextAttemptAt(task data.DataSyncTask, taskErr error) time.Time {
	attempt := task.AttemptCount
	if attempt < 1 {
		attempt = 1
	}
	delay := boundedExponentialBackoff(runner.config.RetryBackoff, runner.config.MaxRetryBackoff, attempt)
	if retryAfter, ok := exchange.RetryAfter(taskErr); ok && retryAfter > delay {
		delay = retryAfter
	}
	return runner.now().Add(delay)
}

func boundedExponentialBackoff(base time.Duration, maxDelay time.Duration, attempt int) time.Duration {
	if attempt <= 1 {
		return base
	}
	delay := base
	for current := 1; current < attempt; current++ {
		if delay >= maxDelay/2 {
			return maxDelay
		}
		delay *= 2
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

type syncWindow struct {
	from time.Time
	to   time.Time
}

func (runner *Runner) syncWindow(task data.DataSyncTask, interval time.Duration) syncWindow {
	now := runner.now()
	to := now
	if task.EndTime != nil && task.EndTime.Before(to) {
		to = *task.EndTime
	}

	from := to.Add(-runner.config.DefaultLookback)
	if task.StartTime != nil {
		from = *task.StartTime
	}
	if task.LatestSyncedOpenTime != nil {
		from = task.LatestSyncedOpenTime.Add(-time.Duration(runner.config.OverlapCandles) * interval)
		if task.StartTime != nil && from.Before(*task.StartTime) {
			from = *task.StartTime
		}
	}

	return syncWindow{from: from.UTC(), to: to.UTC()}
}

func (runner *Runner) isCompleted(
	task data.DataSyncTask,
	interval time.Duration,
	cursorOpenTime *time.Time,
	emptyBatch bool,
	hasClosedCandles bool,
) bool {
	if task.RealtimeEnabled {
		return false
	}
	if task.EndTime == nil {
		return emptyBatch || cursorOpenTime != nil || hasClosedCandles
	}
	if cursorOpenTime == nil {
		return emptyBatch
	}
	nextOpen := cursorOpenTime.Add(interval)
	return !nextOpen.Before(task.EndTime.UTC())
}

func isAlreadySyncedThroughEnd(task data.DataSyncTask, interval time.Duration) bool {
	if task.RealtimeEnabled || task.EndTime == nil || task.LatestSyncedOpenTime == nil {
		return false
	}
	nextOpen := task.LatestSyncedOpenTime.Add(interval)
	return !nextOpen.Before(task.EndTime.UTC())
}

func nextCursorOpenTime(
	task data.DataSyncTask,
	interval time.Duration,
	candles []data.Candle,
) *time.Time {
	ordered := uniqueSortedOpenTimes(candles)
	if len(ordered) == 0 {
		return nil
	}

	chainStart := ordered[0]
	chainTip := chainStart
	for _, openTime := range ordered[1:] {
		expected := chainTip.Add(interval)
		if !openTime.Equal(expected) {
			break
		}
		chainTip = openTime
	}

	if task.LatestSyncedOpenTime == nil {
		return ptrTime(chainTip)
	}
	current := task.LatestSyncedOpenTime.UTC()
	if !chainTip.After(current) {
		return nil
	}
	if chainStart.After(current.Add(interval)) {
		return nil
	}
	return ptrTime(chainTip)
}

func uniqueSortedOpenTimes(candles []data.Candle) []time.Time {
	openTimes := make([]time.Time, 0, len(candles))
	seen := make(map[time.Time]struct{}, len(candles))
	for _, candle := range candles {
		openTime := candle.OpenTime.UTC()
		if _, exists := seen[openTime]; exists {
			continue
		}
		seen[openTime] = struct{}{}
		openTimes = append(openTimes, openTime)
	}
	sort.Slice(openTimes, func(left int, right int) bool {
		return openTimes[left].Before(openTimes[right])
	})
	return openTimes
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
