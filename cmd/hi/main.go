package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: hi <api|migrate>")
}
