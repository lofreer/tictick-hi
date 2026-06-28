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
	Interval    time.Duration
	SyncOnStart bool
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
		instruments, err := client.FetchInstruments(ctx)
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
			)
		}
		results = append(results, result)
	}
	return results
}
