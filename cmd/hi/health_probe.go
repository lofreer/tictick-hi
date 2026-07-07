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
	Command         string
	Addr            string
	WorkerID        string
	StartedAt       time.Time
	ReadinessChecks []workerReadinessCheck
}

type workerReadinessCheck struct {
	Name  string
	Check func(context.Context) error
}

type workerHealthProbeResponse struct {
	Status        string            `json:"status"`
	Command       string            `json:"command"`
	WorkerID      string            `json:"workerId,omitempty"`
	UptimeSeconds int64             `json:"uptimeSeconds,omitempty"`
	Checks        map[string]string `json:"checks,omitempty"`
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
	if config.StartedAt.IsZero() {
		config.StartedAt = time.Now().UTC()
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

func startConfiguredWorkerHealthProbe(
	ctx context.Context,
	command string,
	addr string,
	workerID string,
	readinessChecks []workerReadinessCheck,
) error {
	_, err := startWorkerHealthProbe(ctx, workerHealthProbeConfig{
		Command:         command,
		Addr:            addr,
		WorkerID:        workerID,
		ReadinessChecks: readinessChecks,
	})
	return err
}

func newWorkerHealthProbeHandler(config workerHealthProbeConfig) http.Handler {
	if config.StartedAt.IsZero() {
		config.StartedAt = time.Now().UTC()
	}
	mux := http.NewServeMux()
	handler := func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			response.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payload := workerHealthProbeResponse{
			Status:        "ok",
			Command:       config.Command,
			WorkerID:      config.WorkerID,
			UptimeSeconds: int64(time.Since(config.StartedAt).Seconds()),
		}
		statusCode := http.StatusOK
		if isWorkerReadinessPath(request.URL.Path) && len(config.ReadinessChecks) > 0 {
			payload.Checks = make(map[string]string, len(config.ReadinessChecks))
			for _, check := range config.ReadinessChecks {
				if check.Name == "" || check.Check == nil {
					continue
				}
				payload.Checks[check.Name] = "ok"
				if err := check.Check(request.Context()); err != nil {
					payload.Status = "unavailable"
					payload.Checks[check.Name] = "unavailable"
					statusCode = http.StatusServiceUnavailable
				}
			}
		}
		response.Header().Set("Content-Type", "application/json")
		response.Header().Set("Cache-Control", "no-store")
		response.WriteHeader(statusCode)
		if request.Method == http.MethodHead {
			return
		}
		_ = json.NewEncoder(response).Encode(payload)
	}
	mux.HandleFunc("/livez", handler)
	mux.HandleFunc("/readyz", handler)
	mux.HandleFunc("/healthz", handler)
	return mux
}

func isWorkerReadinessPath(path string) bool {
	return path == "/readyz" || path == "/healthz"
}
