package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type workerHealthProbeConfig struct {
	Command  string
	Addr     string
	WorkerID string
}

func loadWorkerHealthProbeAddr(command string) (string, error) {
	key := strings.ToUpper(command) + "_HEALTH_ADDR"
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", nil
	}
	if err := validateWorkerHealthProbeAddr(key, value); err != nil {
		return "", err
	}
	return value, nil
}

func validateWorkerHealthProbeAddr(key string, value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("%s must be a host:port address", key)
	}
	parsedPort, err := strconv.Atoi(port)
	if err != nil || parsedPort < 0 || parsedPort > 65535 {
		return fmt.Errorf("%s must include a numeric TCP port", key)
	}
	if strings.Contains(host, "/") {
		return fmt.Errorf("%s must be a TCP host:port address", key)
	}
	return nil
}

func startWorkerHealthProbe(ctx context.Context, config workerHealthProbeConfig) (string, error) {
	if config.Addr == "" {
		return "", nil
	}
	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return "", fmt.Errorf("start %s health probe on %s: %w", config.Command, config.Addr, err)
	}

	server := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           newWorkerHealthProbeHandler(config),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("worker health probe shutdown failed", "command", config.Command, "error", err)
		}
	}()

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("worker health probe stopped", "command", config.Command, "addr", server.Addr, "error", err)
		}
	}()

	slog.Info("started worker health probe", "command", config.Command, "addr", server.Addr)
	return server.Addr, nil
}

func startConfiguredWorkerHealthProbe(ctx context.Context, command string, addr string, workerID string) error {
	_, err := startWorkerHealthProbe(ctx, workerHealthProbeConfig{
		Command:  command,
		Addr:     addr,
		WorkerID: workerID,
	})
	return err
}

func newWorkerHealthProbeHandler(config workerHealthProbeConfig) http.Handler {
	mux := http.NewServeMux()
	handler := func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			response.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		response.Header().Set("Cache-Control", "no-store")
		response.WriteHeader(http.StatusOK)
		if request.Method == http.MethodHead {
			return
		}
		_ = json.NewEncoder(response).Encode(map[string]string{
			"status":   "ok",
			"command":  config.Command,
			"workerId": config.WorkerID,
		})
	}
	mux.HandleFunc("/livez", handler)
	mux.HandleFunc("/readyz", handler)
	mux.HandleFunc("/healthz", handler)
	return mux
}
