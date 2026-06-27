package main

import (
	"context"
	"flag"
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
	flags := flag.NewFlagSet("notify", flag.ContinueOnError)
	once := flags.Bool("once", false, "run one claim cycle and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	runner := notification.NewRunner(store, notification.DemoProviders(), notification.Config{
		WorkerID:      envOrDefault("NOTIFY_WORKER_ID", defaultWorkerID()),
		LeaseTTL:      durationEnv("NOTIFY_LEASE_TTL", 30*time.Second),
		PollInterval:  durationEnv("NOTIFY_POLL_INTERVAL", 10*time.Second),
		RetryDelay:    durationEnv("NOTIFY_RETRY_DELAY", 30*time.Second),
		MaxRetryDelay: durationEnv("NOTIFY_MAX_RETRY_DELAY", 5*time.Minute),
	})

	if *once {
		return runner.RunOnce(ctx)
	}
	return runner.Run(ctx)
}

func runTrading(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("trading", flag.ContinueOnError)
	once := flags.Bool("once", false, "run one claim cycle and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	runner := trading.NewRunner(store, strategy.BuiltinRegistry(), trading.Config{
		WorkerID:     envOrDefault("TRADING_WORKER_ID", defaultWorkerID()),
		LeaseTTL:     durationEnv("TRADING_LEASE_TTL", 30*time.Second),
		PollInterval: durationEnv("TRADING_POLL_INTERVAL", 10*time.Second),
		CandleLimit:  intEnv("TRADING_CANDLE_LIMIT", 500),
	})

	if *once {
		return runner.RunOnce(ctx)
	}
	return runner.Run(ctx)
}

func runBacktest(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("backtest", flag.ContinueOnError)
	once := flags.Bool("once", false, "run one claim cycle and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	runner := backtest.NewRunner(store, strategy.BuiltinRegistry(), backtest.Config{
		WorkerID:     envOrDefault("BACKTEST_WORKER_ID", defaultWorkerID()),
		LeaseTTL:     durationEnv("BACKTEST_LEASE_TTL", 30*time.Second),
		PollInterval: durationEnv("BACKTEST_POLL_INTERVAL", 10*time.Second),
		CandleLimit:  intEnv("BACKTEST_CANDLE_LIMIT", 5000),
	})

	if *once {
		return runner.RunOnce(ctx)
	}
	return runner.Run(ctx)
}

func runSync(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	once := flags.Bool("once", false, "run one claim cycle and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	runner := datasync.NewRunner(store, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": binance.NewMarketClientWithBaseURLs(stringListEnv("BINANCE_BASE_URLS"), nil),
		"okx":     okx.NewMarketClient(nil),
	}), datasync.Config{
		WorkerID:       envOrDefault("SYNC_WORKER_ID", defaultWorkerID()),
		LeaseTTL:       durationEnv("SYNC_LEASE_TTL", 30*time.Second),
		PollInterval:   durationEnv("SYNC_POLL_INTERVAL", 10*time.Second),
		BatchLimit:     intEnv("SYNC_BATCH_LIMIT", 500),
		OverlapCandles: intEnv("SYNC_OVERLAP_CANDLES", 2),
		DefaultLookback: durationEnv(
			"SYNC_DEFAULT_LOOKBACK",
			500*time.Minute,
		),
		FetchRetries: intEnv("SYNC_FETCH_RETRIES", 2),
		RetryDelay:   durationEnv("SYNC_RETRY_DELAY", 250*time.Millisecond),
	})

	if *once {
		return runner.RunOnce(ctx)
	}
	return runner.Run(ctx)
}

func runAPI(ctx context.Context) error {
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return err
	}

	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := bootstrapOperator(ctx, store); err != nil {
		return err
	}

	addr := envOrDefault("HTTP_ADDR", "127.0.0.1:8080")
	staticRoot := envOrDefault("WEB_FRONTEND_DIST", "web/frontend/dist")
	server := &http.Server{
		Addr: addr,
		Handler: webapi.NewServerWithConfig(store, webapi.Config{
			StaticRoot:   staticRoot,
			SessionTTL:   durationEnv("AUTH_SESSION_TTL", 12*time.Hour),
			CookieSecure: boolEnv("AUTH_COOKIE_SECURE", false),
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

	slog.Info("starting api", "addr", addr, "static_root", staticRoot)
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

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func intEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func boolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
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
