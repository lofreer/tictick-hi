package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lofreer/tictick-hi/internal/adapter/binance"
	"github.com/lofreer/tictick-hi/internal/adapter/okx"
	"github.com/lofreer/tictick-hi/internal/backtest"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/datasync"
	"github.com/lofreer/tictick-hi/internal/exchange"
	"github.com/lofreer/tictick-hi/internal/marketsync"
	"github.com/lofreer/tictick-hi/internal/notification"
	"github.com/lofreer/tictick-hi/internal/store/postgres"
	"github.com/lofreer/tictick-hi/internal/strategy"
	"github.com/lofreer/tictick-hi/internal/trading"
	webapi "github.com/lofreer/tictick-hi/internal/web/api"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	switch os.Args[1] {
	case "api":
		err = runAPI(ctx)
	case "sync":
		err = runSync(ctx, os.Args[2:])
	case "backtest":
		err = runBacktest(ctx, os.Args[2:])
	case "trading":
		err = runTrading(ctx, os.Args[2:])
	case "notify":
		err = runNotify(ctx, os.Args[2:])
	case "migrate":
		err = runMigrate(ctx)
	case "help", "-h", "--help":
		printUsage()
	default:
		err = fmt.Errorf("unknown subcommand %q", os.Args[1])
	}

	if err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func runNotify(ctx context.Context, args []string) error {
	config, err := loadNotifyCommandConfig(args)
	if err != nil {
		return err
	}
	healthAddr, err := loadWorkerHealthProbeAddr("notify")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, config.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	runner := notification.NewRunner(store, notification.DefaultProviders(), notification.Config{
		WorkerID:      config.WorkerID,
		LeaseTTL:      config.LeaseTTL,
		PollInterval:  config.PollInterval,
		RetryDelay:    config.RetryDelay,
		MaxRetryDelay: config.MaxRetryDelay,
	})

	slog.Info("starting notify", safeConfigSummary(
		"worker_id", config.WorkerID,
		"once", config.Once,
		"lease_ttl", config.LeaseTTL,
		"poll_interval", config.PollInterval,
		"retry_delay", config.RetryDelay,
		"max_retry_delay", config.MaxRetryDelay,
		"health_probe_addr", healthAddr,
	)...)

	if config.Once {
		return runner.RunOnce(ctx)
	}
	if err := startConfiguredWorkerHealthProbe(ctx, "notify", healthAddr, config.WorkerID); err != nil {
		return err
	}
	return runner.Run(ctx)
}

func runTrading(ctx context.Context, args []string) error {
	config, err := loadTradingCommandConfig(args)
	if err != nil {
		return err
	}
	healthAddr, err := loadWorkerHealthProbeAddr("trading")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, config.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	runner := trading.NewRunner(store, strategy.BuiltinRegistry(), trading.Config{
		WorkerID:     config.WorkerID,
		LeaseTTL:     config.LeaseTTL,
		PollInterval: config.PollInterval,
		CandleLimit:  config.CandleLimit,
	})

	slog.Info("starting trading", safeConfigSummary(
		"worker_id", config.WorkerID,
		"once", config.Once,
		"lease_ttl", config.LeaseTTL,
		"poll_interval", config.PollInterval,
		"candle_limit", config.CandleLimit,
		"health_probe_addr", healthAddr,
	)...)

	if config.Once {
		return runner.RunOnce(ctx)
	}
	if err := startConfiguredWorkerHealthProbe(ctx, "trading", healthAddr, config.WorkerID); err != nil {
		return err
	}
	return runner.Run(ctx)
}

func runBacktest(ctx context.Context, args []string) error {
	config, err := loadBacktestCommandConfig(args)
	if err != nil {
		return err
	}
	healthAddr, err := loadWorkerHealthProbeAddr("backtest")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, config.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	runner := backtest.NewRunner(store, strategy.BuiltinRegistry(), backtest.Config{
		WorkerID:     config.WorkerID,
		LeaseTTL:     config.LeaseTTL,
		PollInterval: config.PollInterval,
		CandleLimit:  config.CandleLimit,
	})

	slog.Info("starting backtest", safeConfigSummary(
		"worker_id", config.WorkerID,
		"once", config.Once,
		"lease_ttl", config.LeaseTTL,
		"poll_interval", config.PollInterval,
		"candle_limit", config.CandleLimit,
		"health_probe_addr", healthAddr,
	)...)

	if config.Once {
		return runner.RunOnce(ctx)
	}
	if err := startConfiguredWorkerHealthProbe(ctx, "backtest", healthAddr, config.WorkerID); err != nil {
		return err
	}
	return runner.Run(ctx)
}

func runSync(ctx context.Context, args []string) error {
	config, err := loadSyncCommandConfig(args)
	if err != nil {
		return err
	}
	healthAddr, err := loadWorkerHealthProbeAddr("sync")
	if err != nil {
		return err
	}
	exchangeConfig, err := loadExchangeClientConfig()
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, config.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	binanceClient := newBinanceMarketClient(exchangeConfig)
	okxClient := newOKXMarketClient(exchangeConfig)
	runner := datasync.NewRunner(store, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": binanceClient,
		"okx":     okxClient,
	}), datasync.Config{
		WorkerID:          config.WorkerID,
		LeaseTTL:          config.LeaseTTL,
		HeartbeatInterval: config.HeartbeatInterval,
		PollInterval:      config.PollInterval,
		BatchLimit:        config.BatchLimit,
		OverlapCandles:    config.OverlapCandles,
		DefaultLookback:   config.DefaultLookback,
		FetchRetries:      config.FetchRetries,
		RetryDelay:        config.RetryDelay,
		RetryBackoff:      config.RetryBackoff,
		MaxRetryBackoff:   config.MaxRetryBackoff,
	})

	slog.Info("starting sync", safeConfigSummary(
		"worker_id", config.WorkerID,
		"once", config.Once,
		"lease_ttl", config.LeaseTTL,
		"heartbeat_interval", config.HeartbeatInterval,
		"poll_interval", config.PollInterval,
		"batch_limit", config.BatchLimit,
		"overlap_candles", config.OverlapCandles,
		"default_lookback", config.DefaultLookback,
		"fetch_retries", config.FetchRetries,
		"retry_delay", config.RetryDelay,
		"retry_backoff", config.RetryBackoff,
		"max_retry_backoff", config.MaxRetryBackoff,
		"market_instrument_sync_enabled", config.MarketInstrumentSyncEnabled,
		"market_instrument_sync_interval", config.MarketInstrumentSyncInterval,
		"market_instrument_sync_on_start", config.MarketInstrumentSyncOnStart,
		"binance_request_weight_limit", exchangeConfig.BinanceRequestWeightLimit,
		"binance_request_weight_window", exchangeConfig.BinanceRequestWeightWindow,
		"okx_market_request_limit", exchangeConfig.OKXMarketRequestLimit,
		"okx_market_request_window", exchangeConfig.OKXMarketRequestWindow,
		"health_probe_addr", healthAddr,
	)...)

	if config.Once {
		return runner.RunOnce(ctx)
	}
	if err := startConfiguredWorkerHealthProbe(ctx, "sync", healthAddr, config.WorkerID); err != nil {
		return err
	}
	if config.MarketInstrumentSyncEnabled {
		instrumentRunner := marketsync.NewRunner(store, map[string]exchange.InstrumentClient{
			"binance": binanceClient,
			"okx":     okxClient,
		}, marketsync.Config{
			Interval:     config.MarketInstrumentSyncInterval,
			SyncOnStart:  config.MarketInstrumentSyncOnStart,
			FetchRetries: config.FetchRetries,
			RetryDelay:   config.RetryDelay,
		})
		go func() {
			if err := instrumentRunner.Run(ctx); err != nil {
				slog.Error("market instrument sync runner stopped", "error", err)
			}
		}()
	}
	return runner.Run(ctx)
}

func runAPI(ctx context.Context) error {
	config, err := loadAPICommandConfig()
	if err != nil {
		return err
	}
	exchangeConfig, err := loadExchangeClientConfig()
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, config.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := bootstrapOperator(ctx, store); err != nil {
		return err
	}

	server := &http.Server{
		Addr: config.Addr,
		Handler: webapi.NewServerWithConfig(store, webapi.Config{
			StaticRoot:   config.StaticRoot,
			SessionTTL:   config.SessionTTL,
			CookieSecure: config.CookieSecure,
			InstrumentClients: map[string]exchange.InstrumentClient{
				"binance": newBinanceMarketClient(exchangeConfig),
				"okx":     newOKXMarketClient(exchangeConfig),
			},
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("api shutdown failed", "error", err)
		}
	}()

	slog.Info("starting api", safeConfigSummary(
		"addr", config.Addr,
		"static_root", config.StaticRoot,
		"session_ttl", config.SessionTTL,
		"cookie_secure", config.CookieSecure,
		"binance_request_weight_limit", exchangeConfig.BinanceRequestWeightLimit,
		"binance_request_weight_window", exchangeConfig.BinanceRequestWeightWindow,
		"okx_market_request_limit", exchangeConfig.OKXMarketRequestLimit,
		"okx_market_request_window", exchangeConfig.OKXMarketRequestWindow,
	)...)
	err = server.ListenAndServe()
	if err == nil || err == http.ErrServerClosed {
		return nil
	}
	return err
}

func runMigrate(ctx context.Context) error {
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	return store.Migrate(ctx)
}

func bootstrapOperator(ctx context.Context, store *postgres.Store) error {
	username := strings.TrimSpace(os.Getenv("BOOTSTRAP_OPERATOR_USERNAME"))
	password := os.Getenv("BOOTSTRAP_OPERATOR_PASSWORD")
	if username == "" && password == "" {
		return nil
	}
	if username == "" || password == "" {
		return fmt.Errorf("BOOTSTRAP_OPERATOR_USERNAME and BOOTSTRAP_OPERATOR_PASSWORD must be set together")
	}
	if len(password) < 8 {
		return fmt.Errorf("BOOTSTRAP_OPERATOR_PASSWORD must be at least 8 characters")
	}

	operator, created, err := store.EnsureOperator(ctx, data.CreateOperator{
		Username: username,
		Password: password,
		Enabled:  true,
	})
	if err != nil {
		return err
	}
	if created {
		slog.Info("bootstrapped operator", "username", operator.Username)
	}
	return nil
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func newBinanceMarketClient(config exchangeClientConfig) *binance.MarketClient {
	return binance.NewMarketClientWithOptions(binance.MarketClientOptions{
		BaseURLs: config.BinanceBaseURLs,
		RateLimiter: exchange.NewFixedWindowRateLimiter(
			config.BinanceRequestWeightLimit,
			config.BinanceRequestWeightWindow,
		),
	})
}

func newOKXMarketClient(config exchangeClientConfig) *okx.MarketClient {
	return okx.NewMarketClientWithOptions(okx.MarketClientOptions{
		RateLimiter: exchange.NewFixedWindowRateLimiter(
			config.OKXMarketRequestLimit,
			config.OKXMarketRequestWindow,
		),
	})
}

func stringListEnv(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func defaultWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "localhost"
	}
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: hi <api|sync|backtest|trading|notify|migrate>")
}
