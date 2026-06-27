package datasync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
	"github.com/lofreer/tictick-hi/internal/workerlease"
)

type Runner struct {
	repository data.SyncRepository
	registry   exchange.Registry
	config     Config
	now        func() time.Time
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
	if config.WorkerID == "" {
		config.WorkerID = "sync-worker"
	}

	return &Runner{
		repository: repository,
		registry:   registry,
		config:     config,
		now:        func() time.Time { return time.Now().UTC() },
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
			if releaseErr := runner.repository.ReleaseDataSyncTask(releaseCtx, task.ID); releaseErr != nil {
				return fmt.Errorf("release data sync task on shutdown: %w", releaseErr)
			}
			return nil
		}
		slog.Error("data sync task failed", "task_id", task.ID, "error", err)
		if markErr := runner.repository.MarkDataSyncFailed(ctx, task.ID, err); markErr != nil {
			return fmt.Errorf("mark data sync failed: %w", markErr)
		}
	}
	return nil
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
	client, err := runner.registry.Client(task.Exchange)
	if err != nil {
		return err
	}

	duration, err := data.IntervalDuration(task.Interval)
	if err != nil {
		return err
	}

	window := runner.syncWindow(task, duration)
	if !window.from.Before(window.to) {
		return runner.repository.SaveDataSyncResult(ctx, data.DataSyncResult{
			TaskID:    task.ID,
			Completed: !task.RealtimeEnabled,
		})
	}

	candles, err := runner.fetchCandles(ctx, client, exchange.CandleRequest{
		Exchange: task.Exchange,
		Symbol:   task.Symbol,
		Interval: task.Interval,
		From:     window.from,
		To:       window.to,
		Limit:    runner.config.BatchLimit,
	})
	if err != nil {
		return err
	}

	lastOpenTime := latestOpenTime(candles)
	if err := runner.repository.HeartbeatDataSyncTask(ctx, task.ID, runner.config.WorkerID, runner.config.LeaseTTL); err != nil {
		return err
	}
	return runner.repository.SaveDataSyncResult(ctx, data.DataSyncResult{
		TaskID:       task.ID,
		Candles:      candles,
		LastOpenTime: lastOpenTime,
		Completed:    runner.isCompleted(task, duration, candles, window.to),
	})
}

func (runner *Runner) fetchCandles(
	ctx context.Context,
	client exchange.MarketDataClient,
	request exchange.CandleRequest,
) ([]data.Candle, error) {
	attempts := runner.config.FetchRetries + 1
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		candles, err := client.FetchCandles(ctx, request)
		if err == nil {
			return candles, nil
		}
		lastErr = err
		if !exchange.IsTemporaryError(err) || attempt == attempts {
			return nil, err
		}

		slog.Warn(
			"temporary market data fetch failed; retrying",
			"exchange", request.Exchange,
			"symbol", request.Symbol,
			"interval", request.Interval,
			"attempt", attempt,
			"max_attempts", attempts,
			"error", err,
		)
		if err := waitForRetry(ctx, runner.config.RetryDelay*time.Duration(attempt)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
	candles []data.Candle,
	targetEnd time.Time,
) bool {
	if task.RealtimeEnabled {
		return false
	}
	if task.EndTime == nil {
		return true
	}
	if len(candles) < runner.config.BatchLimit {
		return true
	}

	lastOpen := latestOpenTime(candles)
	if lastOpen == nil {
		return true
	}
	nextOpen := lastOpen.Add(interval)
	return !nextOpen.Before(targetEnd)
}

func latestOpenTime(candles []data.Candle) *time.Time {
	var latest *time.Time
	for _, candle := range candles {
		openTime := candle.OpenTime
		if latest == nil || openTime.After(*latest) {
			latest = &openTime
		}
	}
	return latest
}
