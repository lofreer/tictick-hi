package main

import (
	"strings"
	"testing"
	"time"
)

func TestLoadSyncCommandConfigRequiresDatabaseURL(t *testing.T) {
	clearCommandEnv(t)

	_, err := loadSyncCommandConfig(nil)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("expected DATABASE_URL error, got %v", err)
	}
}

func TestLoadSyncCommandConfigRejectsInvalidDuration(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("SYNC_POLL_INTERVAL", "not-a-duration")

	_, err := loadSyncCommandConfig(nil)
	if err == nil || !strings.Contains(err.Error(), "SYNC_POLL_INTERVAL") {
		t.Fatalf("expected SYNC_POLL_INTERVAL error, got %v", err)
	}
}

func TestLoadBacktestCommandConfigRejectsInvalidInteger(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("BACKTEST_CANDLE_LIMIT", "0")

	_, err := loadBacktestCommandConfig(nil)
	if err == nil || !strings.Contains(err.Error(), "BACKTEST_CANDLE_LIMIT") {
		t.Fatalf("expected BACKTEST_CANDLE_LIMIT error, got %v", err)
	}
}

func TestLoadAPICommandConfigRejectsInvalidBool(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("AUTH_COOKIE_SECURE", "sometimes")

	_, err := loadAPICommandConfig()
	if err == nil || !strings.Contains(err.Error(), "AUTH_COOKIE_SECURE") {
		t.Fatalf("expected AUTH_COOKIE_SECURE error, got %v", err)
	}
}

func TestLoadAPICommandConfigLoadsDatabasePoolOptions(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("DB_MAX_CONNS", "17")
	t.Setenv("DB_MIN_CONNS", "2")
	t.Setenv("DB_MAX_CONN_LIFETIME", "45m")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "5m")

	config, err := loadAPICommandConfig()
	if err != nil {
		t.Fatalf("load api config: %v", err)
	}
	if config.DatabasePool.MaxConns != 17 ||
		config.DatabasePool.MinConns != 2 ||
		config.DatabasePool.MaxConnLifetime != 45*time.Minute ||
		config.DatabasePool.MaxConnIdleTime != 5*time.Minute {
		t.Fatalf("unexpected database pool config: %#v", config.DatabasePool)
	}
}

func TestLoadAPICommandConfigLoadsLoginRateLimitOptions(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("AUTH_LOGIN_FAILURE_LIMIT", "3")
	t.Setenv("AUTH_LOGIN_FAILURE_WINDOW", "2m")
	t.Setenv("AUTH_LOGIN_LOCKOUT", "10m")

	config, err := loadAPICommandConfig()
	if err != nil {
		t.Fatalf("load api config: %v", err)
	}
	if config.LoginFailureLimit != 3 ||
		config.LoginFailureWindow != 2*time.Minute ||
		config.LoginLockout != 10*time.Minute {
		t.Fatalf("unexpected login rate limit config: %#v", config)
	}
}

func TestLoadAPICommandConfigRejectsInvalidLoginRateLimit(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("AUTH_LOGIN_FAILURE_LIMIT", "0")

	_, err := loadAPICommandConfig()
	if err == nil || !strings.Contains(err.Error(), "AUTH_LOGIN_FAILURE_LIMIT") {
		t.Fatalf("expected AUTH_LOGIN_FAILURE_LIMIT error, got %v", err)
	}
}

func TestLoadAPICommandConfigRejectsInvalidLoginRateLimitDuration(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("AUTH_LOGIN_FAILURE_WINDOW", "not-a-duration")

	_, err := loadAPICommandConfig()
	if err == nil || !strings.Contains(err.Error(), "AUTH_LOGIN_FAILURE_WINDOW") {
		t.Fatalf("expected AUTH_LOGIN_FAILURE_WINDOW error, got %v", err)
	}
}

func TestLoadAPICommandConfigRejectsInvalidDatabasePool(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("DB_MAX_CONNS", "2")
	t.Setenv("DB_MIN_CONNS", "3")

	_, err := loadAPICommandConfig()
	if err == nil || !strings.Contains(err.Error(), "DB_MIN_CONNS") {
		t.Fatalf("expected DB_MIN_CONNS error, got %v", err)
	}
}

func TestLoadSyncCommandConfigDefaultsHeartbeatFromLeaseTTL(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("SYNC_LEASE_TTL", "45s")

	config, err := loadSyncCommandConfig([]string{"--once"})
	if err != nil {
		t.Fatalf("load sync config: %v", err)
	}
	if !config.Once {
		t.Fatalf("expected once flag")
	}
	if config.HeartbeatInterval != 15*time.Second {
		t.Fatalf("HeartbeatInterval = %s, want 15s", config.HeartbeatInterval)
	}
}

func TestLoadSyncCommandConfigRejectsHeartbeatLongerThanLease(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	t.Setenv("SYNC_LEASE_TTL", "10s")
	t.Setenv("SYNC_HEARTBEAT_INTERVAL", "11s")

	_, err := loadSyncCommandConfig(nil)
	if err == nil || !strings.Contains(err.Error(), "SYNC_HEARTBEAT_INTERVAL") {
		t.Fatalf("expected heartbeat interval error, got %v", err)
	}
}

func TestLoadExchangeClientConfigRejectsInvalidLimit(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("BINANCE_REQUEST_WEIGHT_LIMIT", "-1")

	_, err := loadExchangeClientConfig()
	if err == nil || !strings.Contains(err.Error(), "BINANCE_REQUEST_WEIGHT_LIMIT") {
		t.Fatalf("expected BINANCE_REQUEST_WEIGHT_LIMIT error, got %v", err)
	}
}

func TestSafeConfigSummaryRedactsSensitiveKeys(t *testing.T) {
	summary := safeConfigSummary(
		"addr", "127.0.0.1:8080",
		"database_url", "postgres://secret",
		"bootstrap_password", "secret-password",
		"session_secret", "secret",
		"api_key", "key",
		"encryption_key", "encryption",
		"static_root", "web/frontend/dist",
	)

	if hasSummaryKey(summary, "database_url") ||
		hasSummaryKey(summary, "bootstrap_password") ||
		hasSummaryKey(summary, "session_secret") ||
		hasSummaryKey(summary, "api_key") ||
		hasSummaryKey(summary, "encryption_key") {
		t.Fatalf("summary leaked sensitive keys: %#v", summary)
	}
	if !hasSummaryPair(summary, "addr", "127.0.0.1:8080") ||
		!hasSummaryPair(summary, "static_root", "web/frontend/dist") {
		t.Fatalf("summary dropped safe keys: %#v", summary)
	}
}

func TestDurationEnvStrictNamesInvalidEnv(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("NOTIFY_RETRY_DELAY", "-1s")

	_, err := durationEnvStrict("NOTIFY_RETRY_DELAY", time.Second)
	if err == nil || !strings.Contains(err.Error(), "NOTIFY_RETRY_DELAY") {
		t.Fatalf("expected named duration error, got %v", err)
	}
}

func TestLoadWorkerReadinessBacklogConfig(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("SYNC_READY_MAX_BACKLOG", "12")
	t.Setenv("SYNC_READY_MAX_AGE", "45s")

	config, err := loadWorkerReadinessBacklogConfig("sync")
	if err != nil {
		t.Fatalf("load worker readiness backlog config: %v", err)
	}
	if config.MaxBacklog != 12 || config.MaxAge != 45*time.Second {
		t.Fatalf("unexpected backlog config: %#v", config)
	}
	if !config.enabled() {
		t.Fatalf("expected backlog readiness to be enabled")
	}
	if limits := config.limits(); limits.MaxBacklog != 12 || limits.MaxReadyAge != 45*time.Second {
		t.Fatalf("unexpected backlog limits: %#v", limits)
	}

	clearCommandEnv(t)
	t.Setenv("TRADING_READY_MAX_AGE", "0s")

	config, err = loadWorkerReadinessBacklogConfig("trading")
	if err != nil {
		t.Fatalf("load disabled worker readiness backlog config: %v", err)
	}
	if config.enabled() {
		t.Fatalf("zero backlog config should be disabled: %#v", config)
	}
}

func TestLoadWorkerReadinessBacklogConfigRejectsInvalidValues(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("NOTIFY_READY_MAX_BACKLOG", "-1")

	_, err := loadWorkerReadinessBacklogConfig("notify")
	if err == nil || !strings.Contains(err.Error(), "NOTIFY_READY_MAX_BACKLOG") {
		t.Fatalf("expected max backlog error, got %v", err)
	}

	clearCommandEnv(t)
	t.Setenv("NOTIFY_READY_MAX_AGE", "-1s")

	_, err = loadWorkerReadinessBacklogConfig("notify")
	if err == nil || !strings.Contains(err.Error(), "NOTIFY_READY_MAX_AGE") {
		t.Fatalf("expected max age error, got %v", err)
	}
}

func TestLoadWorkerReadinessStaleLeaseConfig(t *testing.T) {
	clearCommandEnv(t)

	config, err := loadWorkerReadinessStaleLeaseConfig("sync")
	if err != nil {
		t.Fatalf("load disabled stale lease config: %v", err)
	}
	if config.Enabled {
		t.Fatalf("blank stale lease config should be disabled: %#v", config)
	}

	t.Setenv("SYNC_READY_MAX_STALE_LEASES", "0")
	config, err = loadWorkerReadinessStaleLeaseConfig("sync")
	if err != nil {
		t.Fatalf("load stale lease config: %v", err)
	}
	if !config.Enabled || config.MaxStaleLeases != 0 {
		t.Fatalf("unexpected stale lease config: %#v", config)
	}
	if limits := config.limits(); limits.MaxStaleLeases != 0 {
		t.Fatalf("unexpected stale lease limits: %#v", limits)
	}
	if value := config.summaryValue(); value != 0 {
		t.Fatalf("summary value = %#v, want 0", value)
	}
}

func TestLoadWorkerReadinessStaleLeaseConfigRejectsInvalidValues(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("BACKTEST_READY_MAX_STALE_LEASES", "-1")

	_, err := loadWorkerReadinessStaleLeaseConfig("backtest")
	if err == nil || !strings.Contains(err.Error(), "BACKTEST_READY_MAX_STALE_LEASES") {
		t.Fatalf("expected max stale leases error, got %v", err)
	}
}

func TestLoadWorkerReadinessExchangeBackoffConfig(t *testing.T) {
	clearCommandEnv(t)

	config, err := loadWorkerReadinessExchangeBackoffConfig("backtest")
	if err != nil {
		t.Fatalf("load non-sync exchange backoff config: %v", err)
	}
	if config.Enabled {
		t.Fatalf("non-sync exchange backoff config should be disabled: %#v", config)
	}

	t.Setenv("SYNC_READY_MAX_EXCHANGE_BACKOFFS", "0")
	config, err = loadWorkerReadinessExchangeBackoffConfig("sync")
	if err != nil {
		t.Fatalf("load exchange backoff config: %v", err)
	}
	if !config.Enabled || config.MaxActiveBackoffs != 0 {
		t.Fatalf("unexpected exchange backoff config: %#v", config)
	}
	if limits := config.limits(); limits.MaxActiveBackoffs != 0 {
		t.Fatalf("unexpected exchange backoff limits: %#v", limits)
	}
	if value := config.summaryValue(); value != 0 {
		t.Fatalf("summary value = %#v, want 0", value)
	}
}

func TestLoadWorkerReadinessExchangeBackoffConfigRejectsInvalidValues(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("SYNC_READY_MAX_EXCHANGE_BACKOFFS", "-1")

	_, err := loadWorkerReadinessExchangeBackoffConfig("sync")
	if err == nil || !strings.Contains(err.Error(), "SYNC_READY_MAX_EXCHANGE_BACKOFFS") {
		t.Fatalf("expected max exchange backoffs error, got %v", err)
	}
}

func TestLoadWorkerReadinessCatalogFreshnessConfig(t *testing.T) {
	clearCommandEnv(t)

	config, err := loadWorkerReadinessCatalogFreshnessConfig("backtest")
	if err != nil {
		t.Fatalf("load non-sync catalog freshness config: %v", err)
	}
	if config.enabled() {
		t.Fatalf("non-sync catalog freshness config should be disabled: %#v", config)
	}

	t.Setenv("SYNC_READY_MAX_CATALOG_STALENESS", "25h")
	config, err = loadWorkerReadinessCatalogFreshnessConfig("sync")
	if err != nil {
		t.Fatalf("load catalog freshness config: %v", err)
	}
	if !config.enabled() || config.MaxStaleness != 25*time.Hour {
		t.Fatalf("unexpected catalog freshness config: %#v", config)
	}
	if limits := config.limits(); limits.MaxStaleness != 25*time.Hour {
		t.Fatalf("unexpected catalog freshness limits: %#v", limits)
	}
	if value := config.summaryValue(); value != 25*time.Hour {
		t.Fatalf("summary value = %#v, want 25h", value)
	}
}

func TestLoadWorkerReadinessCatalogFreshnessConfigRejectsInvalidValues(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("SYNC_READY_MAX_CATALOG_STALENESS", "0s")

	_, err := loadWorkerReadinessCatalogFreshnessConfig("sync")
	if err == nil || !strings.Contains(err.Error(), "SYNC_READY_MAX_CATALOG_STALENESS") {
		t.Fatalf("expected catalog staleness error, got %v", err)
	}

	clearCommandEnv(t)
	t.Setenv("SYNC_READY_MAX_CATALOG_STALENESS", "stage8_config_secret")

	_, err = loadWorkerReadinessCatalogFreshnessConfig("sync")
	if err == nil || !strings.Contains(err.Error(), "SYNC_READY_MAX_CATALOG_STALENESS") {
		t.Fatalf("expected catalog staleness duration error, got %v", err)
	}
}

func TestLoadWorkerReadinessProviderConfig(t *testing.T) {
	clearCommandEnv(t)

	config, err := loadWorkerReadinessProviderConfig("sync")
	if err != nil {
		t.Fatalf("load non-notify provider config: %v", err)
	}
	if config.Enabled {
		t.Fatalf("non-notify provider config should be disabled: %#v", config)
	}

	t.Setenv("NOTIFY_READY_VALIDATE_PROVIDER_CONFIG", "true")
	config, err = loadWorkerReadinessProviderConfig("notify")
	if err != nil {
		t.Fatalf("load notify provider config: %v", err)
	}
	if !config.Enabled {
		t.Fatalf("expected notify provider config readiness to be enabled")
	}
	if value := config.summaryValue(); value != true {
		t.Fatalf("summary value = %#v, want true", value)
	}
}

func TestLoadWorkerReadinessProviderConfigRejectsInvalidValues(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("NOTIFY_READY_VALIDATE_PROVIDER_CONFIG", "maybe")

	_, err := loadWorkerReadinessProviderConfig("notify")
	if err == nil || !strings.Contains(err.Error(), "NOTIFY_READY_VALIDATE_PROVIDER_CONFIG") {
		t.Fatalf("expected provider config readiness error, got %v", err)
	}
}

func clearCommandEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"DATABASE_URL",
		"LOG_LEVEL",
		"LOG_FORMAT",
		"LOG_CORRELATION_ID",
		"LOG_TRACEPARENT",
		"DB_MAX_CONNS",
		"DB_MIN_CONNS",
		"DB_MAX_CONN_LIFETIME",
		"DB_MAX_CONN_IDLE_TIME",
		"HTTP_ADDR",
		"WEB_FRONTEND_DIST",
		"AUTH_SESSION_TTL",
		"AUTH_COOKIE_SECURE",
		"SYNC_WORKER_ID",
		"SYNC_LEASE_TTL",
		"SYNC_HEARTBEAT_INTERVAL",
		"SYNC_POLL_INTERVAL",
		"SYNC_BATCH_LIMIT",
		"SYNC_OVERLAP_CANDLES",
		"SYNC_DEFAULT_LOOKBACK",
		"SYNC_FETCH_RETRIES",
		"SYNC_RETRY_DELAY",
		"SYNC_RETRY_BACKOFF",
		"SYNC_MAX_RETRY_BACKOFF",
		"SYNC_HEALTH_ADDR",
		"SYNC_READY_MAX_BACKLOG",
		"SYNC_READY_MAX_AGE",
		"SYNC_READY_MAX_STALE_LEASES",
		"SYNC_READY_MAX_EXCHANGE_BACKOFFS",
		"SYNC_READY_MAX_CATALOG_STALENESS",
		"MARKET_INSTRUMENT_SYNC_ENABLED",
		"MARKET_INSTRUMENT_SYNC_ON_START",
		"MARKET_INSTRUMENT_SYNC_INTERVAL",
		"BACKTEST_WORKER_ID",
		"BACKTEST_LEASE_TTL",
		"BACKTEST_POLL_INTERVAL",
		"BACKTEST_CANDLE_LIMIT",
		"BACKTEST_HEALTH_ADDR",
		"BACKTEST_READY_MAX_BACKLOG",
		"BACKTEST_READY_MAX_AGE",
		"BACKTEST_READY_MAX_STALE_LEASES",
		"TRADING_WORKER_ID",
		"TRADING_LEASE_TTL",
		"TRADING_POLL_INTERVAL",
		"TRADING_CANDLE_LIMIT",
		"TRADING_HEALTH_ADDR",
		"TRADING_READY_MAX_BACKLOG",
		"TRADING_READY_MAX_AGE",
		"TRADING_READY_MAX_STALE_LEASES",
		"NOTIFY_WORKER_ID",
		"NOTIFY_LEASE_TTL",
		"NOTIFY_POLL_INTERVAL",
		"NOTIFY_RETRY_DELAY",
		"NOTIFY_MAX_RETRY_DELAY",
		"NOTIFY_HEALTH_ADDR",
		"NOTIFY_READY_MAX_BACKLOG",
		"NOTIFY_READY_MAX_AGE",
		"NOTIFY_READY_MAX_STALE_LEASES",
		"NOTIFY_READY_VALIDATE_PROVIDER_CONFIG",
		"BINANCE_BASE_URLS",
		"BINANCE_REQUEST_WEIGHT_LIMIT",
		"BINANCE_REQUEST_WEIGHT_WINDOW",
		"OKX_MARKET_REQUEST_LIMIT",
		"OKX_MARKET_REQUEST_WINDOW",
	} {
		t.Setenv(key, "")
	}
}

func hasSummaryKey(summary []any, key string) bool {
	for index := 0; index+1 < len(summary); index += 2 {
		if summary[index] == key {
			return true
		}
	}
	return false
}

func hasSummaryPair[T comparable](summary []any, key string, value T) bool {
	for index := 0; index+1 < len(summary); index += 2 {
		if summary[index] != key {
			continue
		}
		typedValue, ok := summary[index+1].(T)
		return ok && typedValue == value
	}
	return false
}

func TestBoolEnvStrictAcceptsKnownValues(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("AUTH_COOKIE_SECURE", "on")

	value, err := boolEnvStrict("AUTH_COOKIE_SECURE", false)
	if err != nil {
		t.Fatalf("bool env strict: %v", err)
	}
	if !value {
		t.Fatalf("expected true")
	}
}

func TestLoadNotifyCommandConfigRejectsFlagErrors(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("DATABASE_URL", "postgres://local")

	_, err := loadNotifyCommandConfig([]string{"--unknown"})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected unknown flag error, got %v", err)
	}
}
