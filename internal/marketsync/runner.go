package marketsync

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

type Repository interface {
	TryLockMarketInstrumentSync(
		ctx context.Context,
		exchange string,
	) (func(context.Context) error, bool, error)
	ReplaceMarketInstruments(
		ctx context.Context,
		exchange string,
		instruments []data.MarketInstrument,
		syncedAt time.Time,
	) (data.MarketInstrumentSyncResult, error)
	RecordMarketInstrumentSyncFailure(
		ctx context.Context,
		exchange string,
		syncErr error,
		attemptedAt time.Time,
	) error
}

type Runner struct {
	repository Repository
	clients    map[string]exchange.InstrumentClient
	exchanges  []string
	config     Config
	now        func() time.Time
}

type Config struct {
	Interval     time.Duration
	SyncOnStart  bool
	FetchRetries int
	RetryDelay   time.Duration
}

type ExchangeResult struct {
	Exchange string
	Result   data.MarketInstrumentSyncResult
	Err      error
	Skipped  bool
}

func NewRunner(
	repository Repository,
	clients map[string]exchange.InstrumentClient,
	config Config,
) *Runner {
	if config.Interval <= 0 {
		config.Interval = 6 * time.Hour
	}
	if config.FetchRetries <= 0 {
		config.FetchRetries = 2
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 250 * time.Millisecond
	}
	clonedClients := make(map[string]exchange.InstrumentClient, len(clients))
	exchanges := make([]string, 0, len(clients))
	for exchangeID, client := range clients {
		if client == nil {
			continue
		}
		clonedClients[exchangeID] = client
		exchanges = append(exchanges, exchangeID)
	}
	sort.Strings(exchanges)

	return &Runner{
		repository: repository,
		clients:    clonedClients,
		exchanges:  exchanges,
		config:     config,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (runner *Runner) Run(ctx context.Context) error {
	if runner.config.SyncOnStart {
		runner.RunOnce(ctx)
	}

	ticker := time.NewTicker(runner.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			runner.RunOnce(ctx)
		}
	}
}

func (runner *Runner) RunOnce(ctx context.Context) []ExchangeResult {
	results := make([]ExchangeResult, 0, len(runner.exchanges))
	for _, exchangeID := range runner.exchanges {
		results = append(results, runner.runExchange(ctx, exchangeID, runner.clients[exchangeID]))
	}
	return results
}

func (runner *Runner) runExchange(
	ctx context.Context,
	exchangeID string,
	client exchange.InstrumentClient,
) ExchangeResult {
	result := ExchangeResult{Exchange: exchangeID}
	unlock, locked, err := runner.repository.TryLockMarketInstrumentSync(ctx, exchangeID)
	if err != nil {
		runner.recordFailure(ctx, exchangeID, err)
		result.Err = err
		slog.Warn("market instrument sync lock failed", "exchange", exchangeID, "error", err)
		return result
	}
	if !locked {
		result.Skipped = true
		slog.Info("market instrument sync skipped; lock held", "exchange", exchangeID)
		return result
	}

	instruments, err := runner.fetchInstruments(ctx, exchangeID, client)
	if err == nil {
		result.Result, err = runner.repository.ReplaceMarketInstruments(
			ctx,
			exchangeID,
			instruments,
			runner.now(),
		)
	}

	if unlockErr := unlockMarketInstrumentSync(unlock); unlockErr != nil {
		slog.Error("market instrument sync unlock failed", "exchange", exchangeID, "error", unlockErr)
		if err == nil {
			result.Err = unlockErr
			return result
		}
	}

	if err != nil {
		result.Err = err
		runner.recordFailure(ctx, exchangeID, err)
		slog.Warn("market instrument sync failed", "exchange", exchangeID, "error", err)
		return result
	}

	slog.Info(
		"market instrument catalog synced",
		"exchange", exchangeID,
		"active", result.Result.ActiveCount,
		"inactive", result.Result.InactiveCount,
		"paused_data_sync_tasks", result.Result.PausedDataSyncTaskCount,
		"restored_data_sync_tasks", result.Result.RestoredDataSyncTaskCount,
	)
	return result
}

func (runner *Runner) recordFailure(ctx context.Context, exchangeID string, err error) {
	if ctx.Err() != nil {
		return
	}
	if recordErr := runner.repository.RecordMarketInstrumentSyncFailure(
		ctx,
		exchangeID,
		err,
		runner.now(),
	); recordErr != nil {
		slog.Error(
			"record market instrument sync failure failed",
			"exchange", exchangeID,
			"error", recordErr,
		)
	}
}

func unlockMarketInstrumentSync(unlock func(context.Context) error) error {
	releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := unlock(releaseCtx); err != nil {
		return err
	}
	return nil
}

func (runner *Runner) fetchInstruments(
	ctx context.Context,
	exchangeID string,
	client exchange.InstrumentClient,
) ([]data.MarketInstrument, error) {
	attempts := runner.config.FetchRetries + 1
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		instruments, err := client.FetchInstruments(ctx)
		if err == nil {
			return instruments, nil
		}
		lastErr = err
		if !exchange.IsTemporaryError(err) || attempt == attempts {
			return nil, err
		}

		slog.Warn(
			"temporary market instrument fetch failed; retrying",
			"exchange", exchangeID,
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
