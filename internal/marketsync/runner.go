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
	ReplaceMarketInstruments(
		ctx context.Context,
		exchange string,
		instruments []data.MarketInstrument,
		syncedAt time.Time,
	) (data.MarketInstrumentSyncResult, error)
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
		client := runner.clients[exchangeID]
		result := ExchangeResult{Exchange: exchangeID}
		instruments, err := runner.fetchInstruments(ctx, exchangeID, client)
		if err == nil {
			result.Result, err = runner.repository.ReplaceMarketInstruments(
				ctx,
				exchangeID,
				instruments,
				runner.now(),
			)
		}
		if err != nil {
			result.Err = err
			slog.Warn("market instrument sync failed", "exchange", exchangeID, "error", err)
		} else {
			slog.Info(
				"market instrument catalog synced",
				"exchange", exchangeID,
				"active", result.Result.ActiveCount,
				"inactive", result.Result.InactiveCount,
				"paused_data_sync_tasks", result.Result.PausedDataSyncTaskCount,
			)
		}
		results = append(results, result)
	}
	return results
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
