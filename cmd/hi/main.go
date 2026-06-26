package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lofreer/tictick-hi/internal/adapter/binance"
	"github.com/lofreer/tictick-hi/internal/adapter/okx"
	"github.com/lofreer/tictick-hi/internal/datasync"
	"github.com/lofreer/tictick-hi/internal/exchange"
	"github.com/lofreer/tictick-hi/internal/store/postgres"
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
		"binance": binance.NewMarketClient(nil),
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

	addr := envOrDefault("HTTP_ADDR", "127.0.0.1:8080")
	staticRoot := envOrDefault("WEB_FRONTEND_DIST", "web/frontend/dist")
	server := &http.Server{
		Addr:              addr,
		Handler:           webapi.NewServer(store, staticRoot),
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

func defaultWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "localhost"
	}
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: hi <api|sync|migrate>")
}
