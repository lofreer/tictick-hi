package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/store/postgres"
)

type workerReadinessBacklogConfig struct {
	MaxBacklog int
	MaxAge     time.Duration
}

type workerReadinessStaleLeaseConfig struct {
	Enabled        bool
	MaxStaleLeases int
}

type workerProbeRuntimeConfig struct {
	HealthAddr  string
	Backlog     workerReadinessBacklogConfig
	StaleLeases workerReadinessStaleLeaseConfig
}

func loadWorkerReadinessBacklogConfig(command string) (workerReadinessBacklogConfig, error) {
	prefix := strings.ToUpper(command)
	maxBacklog, err := intEnvStrict(prefix+"_READY_MAX_BACKLOG", 0, 0)
	if err != nil {
		return workerReadinessBacklogConfig{}, err
	}
	maxAge, err := durationEnvNonNegative(prefix+"_READY_MAX_AGE", 0)
	if err != nil {
		return workerReadinessBacklogConfig{}, err
	}
	return workerReadinessBacklogConfig{
		MaxBacklog: maxBacklog,
		MaxAge:     maxAge,
	}, nil
}

func loadWorkerReadinessStaleLeaseConfig(command string) (workerReadinessStaleLeaseConfig, error) {
	prefix := strings.ToUpper(command)
	maxStaleLeases, enabled, err := optionalNonNegativeIntEnv(prefix + "_READY_MAX_STALE_LEASES")
	if err != nil {
		return workerReadinessStaleLeaseConfig{}, err
	}
	return workerReadinessStaleLeaseConfig{
		Enabled:        enabled,
		MaxStaleLeases: maxStaleLeases,
	}, nil
}

func loadWorkerProbeRuntimeConfig(command string) (workerProbeRuntimeConfig, error) {
	healthAddr, err := loadWorkerHealthProbeAddr(command)
	if err != nil {
		return workerProbeRuntimeConfig{}, err
	}
	backlogReadiness, err := loadWorkerReadinessBacklogConfig(command)
	if err != nil {
		return workerProbeRuntimeConfig{}, err
	}
	staleLeaseReadiness, err := loadWorkerReadinessStaleLeaseConfig(command)
	if err != nil {
		return workerProbeRuntimeConfig{}, err
	}
	return workerProbeRuntimeConfig{
		HealthAddr:  healthAddr,
		Backlog:     backlogReadiness,
		StaleLeases: staleLeaseReadiness,
	}, nil
}

func (config workerReadinessBacklogConfig) enabled() bool {
	return config.MaxBacklog > 0 || config.MaxAge > 0
}

func (config workerReadinessBacklogConfig) limits() postgres.WorkerQueueBacklogLimits {
	return postgres.WorkerQueueBacklogLimits{
		MaxBacklog:  config.MaxBacklog,
		MaxReadyAge: config.MaxAge,
	}
}

func (config workerReadinessStaleLeaseConfig) limits() postgres.WorkerStaleLeaseLimits {
	return postgres.WorkerStaleLeaseLimits{MaxStaleLeases: config.MaxStaleLeases}
}

func (config workerReadinessStaleLeaseConfig) summaryValue() any {
	if !config.Enabled {
		return ""
	}
	return config.MaxStaleLeases
}

func optionalNonNegativeIntEnv(key string) (int, bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, fmt.Errorf("%s must be a valid integer", key)
	}
	if parsed < 0 {
		return 0, false, fmt.Errorf("%s must be greater than or equal to 0", key)
	}
	return parsed, true, nil
}

func workerReadinessChecks(
	store *postgres.Store,
	command string,
	config workerProbeRuntimeConfig,
) []workerReadinessCheck {
	checks := []workerReadinessCheck{
		{
			Name:  "postgres",
			Check: store.Ping,
		},
		{
			Name: "queue",
			Check: func(ctx context.Context) error {
				return store.CheckWorkerQueue(ctx, command)
			},
		},
	}
	if config.Backlog.enabled() {
		checks = append(checks, workerReadinessCheck{
			Name: "queue_backlog",
			Check: func(ctx context.Context) error {
				return store.CheckWorkerQueueBacklog(ctx, command, config.Backlog.limits())
			},
		})
	}
	if config.StaleLeases.Enabled {
		checks = append(checks, workerReadinessCheck{
			Name: "stale_leases",
			Check: func(ctx context.Context) error {
				return store.CheckWorkerStaleLeases(ctx, command, config.StaleLeases.limits())
			},
		})
	}
	return checks
}
