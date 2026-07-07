package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/notification"
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

type workerReadinessExchangeBackoffConfig struct {
	Enabled           bool
	MaxActiveBackoffs int
}

type workerReadinessProviderConfig struct {
	Enabled bool
}

type workerReadinessCatalogFreshnessConfig struct {
	MaxStaleness time.Duration
}

type workerProbeRuntimeConfig struct {
	HealthAddr       string
	Backlog          workerReadinessBacklogConfig
	StaleLeases      workerReadinessStaleLeaseConfig
	ExchangeBackoffs workerReadinessExchangeBackoffConfig
	CatalogFreshness workerReadinessCatalogFreshnessConfig
	ProviderConfig   workerReadinessProviderConfig
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

func loadWorkerReadinessExchangeBackoffConfig(command string) (workerReadinessExchangeBackoffConfig, error) {
	if command != "sync" {
		return workerReadinessExchangeBackoffConfig{}, nil
	}
	maxBackoffs, enabled, err := optionalNonNegativeIntEnv("SYNC_READY_MAX_EXCHANGE_BACKOFFS")
	if err != nil {
		return workerReadinessExchangeBackoffConfig{}, err
	}
	return workerReadinessExchangeBackoffConfig{
		Enabled:           enabled,
		MaxActiveBackoffs: maxBackoffs,
	}, nil
}

func loadWorkerReadinessProviderConfig(command string) (workerReadinessProviderConfig, error) {
	if command != "notify" {
		return workerReadinessProviderConfig{}, nil
	}
	enabled, err := boolEnvStrict("NOTIFY_READY_VALIDATE_PROVIDER_CONFIG", false)
	if err != nil {
		return workerReadinessProviderConfig{}, err
	}
	return workerReadinessProviderConfig{Enabled: enabled}, nil
}

func loadWorkerReadinessCatalogFreshnessConfig(command string) (workerReadinessCatalogFreshnessConfig, error) {
	if command != "sync" {
		return workerReadinessCatalogFreshnessConfig{}, nil
	}
	maxStaleness, err := durationEnvStrict("SYNC_READY_MAX_CATALOG_STALENESS", 0)
	if err != nil {
		return workerReadinessCatalogFreshnessConfig{}, err
	}
	return workerReadinessCatalogFreshnessConfig{MaxStaleness: maxStaleness}, nil
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
	exchangeBackoffReadiness, err := loadWorkerReadinessExchangeBackoffConfig(command)
	if err != nil {
		return workerProbeRuntimeConfig{}, err
	}
	catalogFreshnessReadiness, err := loadWorkerReadinessCatalogFreshnessConfig(command)
	if err != nil {
		return workerProbeRuntimeConfig{}, err
	}
	providerConfigReadiness, err := loadWorkerReadinessProviderConfig(command)
	if err != nil {
		return workerProbeRuntimeConfig{}, err
	}
	return workerProbeRuntimeConfig{
		HealthAddr:       healthAddr,
		Backlog:          backlogReadiness,
		StaleLeases:      staleLeaseReadiness,
		ExchangeBackoffs: exchangeBackoffReadiness,
		CatalogFreshness: catalogFreshnessReadiness,
		ProviderConfig:   providerConfigReadiness,
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

func (config workerReadinessExchangeBackoffConfig) limits() postgres.SyncExchangeBackoffLimits {
	return postgres.SyncExchangeBackoffLimits{MaxActiveBackoffs: config.MaxActiveBackoffs}
}

func (config workerReadinessExchangeBackoffConfig) summaryValue() any {
	if !config.Enabled {
		return ""
	}
	return config.MaxActiveBackoffs
}

func (config workerReadinessCatalogFreshnessConfig) enabled() bool {
	return config.MaxStaleness > 0
}

func (config workerReadinessCatalogFreshnessConfig) limits() postgres.SyncCatalogFreshnessLimits {
	return postgres.SyncCatalogFreshnessLimits{MaxStaleness: config.MaxStaleness}
}

func (config workerReadinessCatalogFreshnessConfig) summaryValue() any {
	if !config.enabled() {
		return ""
	}
	return config.MaxStaleness
}

func (config workerReadinessProviderConfig) summaryValue() any {
	if !config.Enabled {
		return ""
	}
	return true
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
	if config.ExchangeBackoffs.Enabled {
		checks = append(checks, workerReadinessCheck{
			Name: "exchange_backoff",
			Check: func(ctx context.Context) error {
				return store.CheckSyncExchangeBackoffs(ctx, config.ExchangeBackoffs.limits())
			},
		})
	}
	if config.CatalogFreshness.enabled() {
		checks = append(checks, workerReadinessCheck{
			Name: "catalog_freshness",
			Check: func(ctx context.Context) error {
				return store.CheckSyncCatalogFreshness(ctx, config.CatalogFreshness.limits())
			},
		})
	}
	if config.ProviderConfig.Enabled {
		checks = append(checks, workerReadinessCheck{
			Name: "notification_providers",
			Check: func(ctx context.Context) error {
				return checkNotificationProviderConfig(ctx, store)
			},
		})
	}
	return checks
}

func checkNotificationProviderConfig(ctx context.Context, store *postgres.Store) error {
	channels, err := store.ListNotificationChannels(ctx)
	if err != nil {
		return fmt.Errorf("read notification provider config: %w", err)
	}
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}
		if err := notification.ValidateProviderTarget(channel.Provider, channel.Target); err != nil {
			return fmt.Errorf(
				"notification channel %q provider %q target invalid: %w",
				channel.Name,
				channel.Provider,
				err,
			)
		}
	}
	return nil
}
