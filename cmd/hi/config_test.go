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

func clearCommandEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"DATABASE_URL",
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
		"MARKET_INSTRUMENT_SYNC_ENABLED",
		"MARKET_INSTRUMENT_SYNC_ON_START",
		"MARKET_INSTRUMENT_SYNC_INTERVAL",
		"BACKTEST_WORKER_ID",
		"BACKTEST_LEASE_TTL",
		"BACKTEST_POLL_INTERVAL",
		"BACKTEST_CANDLE_LIMIT",
		"TRADING_WORKER_ID",
		"TRADING_LEASE_TTL",
		"TRADING_POLL_INTERVAL",
		"TRADING_CANDLE_LIMIT",
		"NOTIFY_WORKER_ID",
		"NOTIFY_LEASE_TTL",
		"NOTIFY_POLL_INTERVAL",
		"NOTIFY_RETRY_DELAY",
		"NOTIFY_MAX_RETRY_DELAY",
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
