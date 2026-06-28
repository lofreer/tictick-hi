package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type apiCommandConfig struct {
	DatabaseURL  string
	Addr         string
	StaticRoot   string
	SessionTTL   time.Duration
	CookieSecure bool
}

type syncCommandConfig struct {
	DatabaseURL                  string
	Once                         bool
	WorkerID                     string
	LeaseTTL                     time.Duration
	HeartbeatInterval            time.Duration
	PollInterval                 time.Duration
	BatchLimit                   int
	OverlapCandles               int
	DefaultLookback              time.Duration
	FetchRetries                 int
	RetryDelay                   time.Duration
	RetryBackoff                 time.Duration
	MaxRetryBackoff              time.Duration
	MarketInstrumentSyncEnabled  bool
	MarketInstrumentSyncInterval time.Duration
	MarketInstrumentSyncOnStart  bool
}

type backtestCommandConfig struct {
	DatabaseURL  string
	Once         bool
	WorkerID     string
	LeaseTTL     time.Duration
	PollInterval time.Duration
	CandleLimit  int
}

type tradingCommandConfig struct {
	DatabaseURL  string
	Once         bool
	WorkerID     string
	LeaseTTL     time.Duration
	PollInterval time.Duration
	CandleLimit  int
}

type notifyCommandConfig struct {
	DatabaseURL   string
	Once          bool
	WorkerID      string
	LeaseTTL      time.Duration
	PollInterval  time.Duration
	RetryDelay    time.Duration
	MaxRetryDelay time.Duration
}

type exchangeClientConfig struct {
	BinanceBaseURLs            []string
	BinanceRequestWeightLimit  int
	BinanceRequestWeightWindow time.Duration
	OKXMarketRequestLimit      int
	OKXMarketRequestWindow     time.Duration
}

func loadAPICommandConfig() (apiCommandConfig, error) {
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return apiCommandConfig{}, err
	}
	sessionTTL, err := durationEnvStrict("AUTH_SESSION_TTL", 12*time.Hour)
	if err != nil {
		return apiCommandConfig{}, err
	}
	cookieSecure, err := boolEnvStrict("AUTH_COOKIE_SECURE", false)
	if err != nil {
		return apiCommandConfig{}, err
	}
	return apiCommandConfig{
		DatabaseURL:  databaseURL,
		Addr:         envOrDefault("HTTP_ADDR", "127.0.0.1:8080"),
		StaticRoot:   envOrDefault("WEB_FRONTEND_DIST", "web/frontend/dist"),
		SessionTTL:   sessionTTL,
		CookieSecure: cookieSecure,
	}, nil
}

func loadSyncCommandConfig(args []string) (syncCommandConfig, error) {
	flags := newCommandFlagSet("sync")
	once := flags.Bool("once", false, "run one claim cycle and exit")
	if err := flags.Parse(args); err != nil {
		return syncCommandConfig{}, err
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return syncCommandConfig{}, err
	}
	leaseTTL, err := durationEnvStrict("SYNC_LEASE_TTL", 30*time.Second)
	if err != nil {
		return syncCommandConfig{}, err
	}
	heartbeatInterval, err := durationEnvStrict("SYNC_HEARTBEAT_INTERVAL", leaseTTL/3)
	if err != nil {
		return syncCommandConfig{}, err
	}
	if heartbeatInterval > leaseTTL {
		return syncCommandConfig{}, fmt.Errorf("SYNC_HEARTBEAT_INTERVAL must be less than or equal to SYNC_LEASE_TTL")
	}
	pollInterval, err := durationEnvStrict("SYNC_POLL_INTERVAL", 10*time.Second)
	if err != nil {
		return syncCommandConfig{}, err
	}
	batchLimit, err := intEnvStrict("SYNC_BATCH_LIMIT", 500, 1)
	if err != nil {
		return syncCommandConfig{}, err
	}
	overlapCandles, err := intEnvStrict("SYNC_OVERLAP_CANDLES", 2, 0)
	if err != nil {
		return syncCommandConfig{}, err
	}
	defaultLookback, err := durationEnvStrict("SYNC_DEFAULT_LOOKBACK", 500*time.Minute)
	if err != nil {
		return syncCommandConfig{}, err
	}
	fetchRetries, err := intEnvStrict("SYNC_FETCH_RETRIES", 2, 1)
	if err != nil {
		return syncCommandConfig{}, err
	}
	retryDelay, err := durationEnvStrict("SYNC_RETRY_DELAY", 250*time.Millisecond)
	if err != nil {
		return syncCommandConfig{}, err
	}
	retryBackoff, err := durationEnvStrict("SYNC_RETRY_BACKOFF", 30*time.Second)
	if err != nil {
		return syncCommandConfig{}, err
	}
	maxRetryBackoff, err := durationEnvStrict("SYNC_MAX_RETRY_BACKOFF", 5*time.Minute)
	if err != nil {
		return syncCommandConfig{}, err
	}
	instrumentSyncEnabled, err := boolEnvStrict("MARKET_INSTRUMENT_SYNC_ENABLED", true)
	if err != nil {
		return syncCommandConfig{}, err
	}
	instrumentSyncOnStart, err := boolEnvStrict("MARKET_INSTRUMENT_SYNC_ON_START", true)
	if err != nil {
		return syncCommandConfig{}, err
	}
	instrumentSyncInterval, err := durationEnvStrict("MARKET_INSTRUMENT_SYNC_INTERVAL", 6*time.Hour)
	if err != nil {
		return syncCommandConfig{}, err
	}

	return syncCommandConfig{
		DatabaseURL:                  databaseURL,
		Once:                         *once,
		WorkerID:                     envOrDefault("SYNC_WORKER_ID", defaultWorkerID()),
		LeaseTTL:                     leaseTTL,
		HeartbeatInterval:            heartbeatInterval,
		PollInterval:                 pollInterval,
		BatchLimit:                   batchLimit,
		OverlapCandles:               overlapCandles,
		DefaultLookback:              defaultLookback,
		FetchRetries:                 fetchRetries,
		RetryDelay:                   retryDelay,
		RetryBackoff:                 retryBackoff,
		MaxRetryBackoff:              maxRetryBackoff,
		MarketInstrumentSyncEnabled:  instrumentSyncEnabled,
		MarketInstrumentSyncInterval: instrumentSyncInterval,
		MarketInstrumentSyncOnStart:  instrumentSyncOnStart,
	}, nil
}

func loadBacktestCommandConfig(args []string) (backtestCommandConfig, error) {
	flags := newCommandFlagSet("backtest")
	once := flags.Bool("once", false, "run one claim cycle and exit")
	if err := flags.Parse(args); err != nil {
		return backtestCommandConfig{}, err
	}
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return backtestCommandConfig{}, err
	}
	leaseTTL, err := durationEnvStrict("BACKTEST_LEASE_TTL", 30*time.Second)
	if err != nil {
		return backtestCommandConfig{}, err
	}
	pollInterval, err := durationEnvStrict("BACKTEST_POLL_INTERVAL", 10*time.Second)
	if err != nil {
		return backtestCommandConfig{}, err
	}
	candleLimit, err := intEnvStrict("BACKTEST_CANDLE_LIMIT", 5000, 1)
	if err != nil {
		return backtestCommandConfig{}, err
	}
	return backtestCommandConfig{
		DatabaseURL:  databaseURL,
		Once:         *once,
		WorkerID:     envOrDefault("BACKTEST_WORKER_ID", defaultWorkerID()),
		LeaseTTL:     leaseTTL,
		PollInterval: pollInterval,
		CandleLimit:  candleLimit,
	}, nil
}

func loadTradingCommandConfig(args []string) (tradingCommandConfig, error) {
	flags := newCommandFlagSet("trading")
	once := flags.Bool("once", false, "run one claim cycle and exit")
	if err := flags.Parse(args); err != nil {
		return tradingCommandConfig{}, err
	}
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return tradingCommandConfig{}, err
	}
	leaseTTL, err := durationEnvStrict("TRADING_LEASE_TTL", 30*time.Second)
	if err != nil {
		return tradingCommandConfig{}, err
	}
	pollInterval, err := durationEnvStrict("TRADING_POLL_INTERVAL", 10*time.Second)
	if err != nil {
		return tradingCommandConfig{}, err
	}
	candleLimit, err := intEnvStrict("TRADING_CANDLE_LIMIT", 500, 1)
	if err != nil {
		return tradingCommandConfig{}, err
	}
	return tradingCommandConfig{
		DatabaseURL:  databaseURL,
		Once:         *once,
		WorkerID:     envOrDefault("TRADING_WORKER_ID", defaultWorkerID()),
		LeaseTTL:     leaseTTL,
		PollInterval: pollInterval,
		CandleLimit:  candleLimit,
	}, nil
}

func loadNotifyCommandConfig(args []string) (notifyCommandConfig, error) {
	flags := newCommandFlagSet("notify")
	once := flags.Bool("once", false, "run one claim cycle and exit")
	if err := flags.Parse(args); err != nil {
		return notifyCommandConfig{}, err
	}
	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return notifyCommandConfig{}, err
	}
	leaseTTL, err := durationEnvStrict("NOTIFY_LEASE_TTL", 30*time.Second)
	if err != nil {
		return notifyCommandConfig{}, err
	}
	pollInterval, err := durationEnvStrict("NOTIFY_POLL_INTERVAL", 10*time.Second)
	if err != nil {
		return notifyCommandConfig{}, err
	}
	retryDelay, err := durationEnvStrict("NOTIFY_RETRY_DELAY", 30*time.Second)
	if err != nil {
		return notifyCommandConfig{}, err
	}
	maxRetryDelay, err := durationEnvStrict("NOTIFY_MAX_RETRY_DELAY", 5*time.Minute)
	if err != nil {
		return notifyCommandConfig{}, err
	}
	return notifyCommandConfig{
		DatabaseURL:   databaseURL,
		Once:          *once,
		WorkerID:      envOrDefault("NOTIFY_WORKER_ID", defaultWorkerID()),
		LeaseTTL:      leaseTTL,
		PollInterval:  pollInterval,
		RetryDelay:    retryDelay,
		MaxRetryDelay: maxRetryDelay,
	}, nil
}

func loadExchangeClientConfig() (exchangeClientConfig, error) {
	binanceLimit, err := intEnvStrict("BINANCE_REQUEST_WEIGHT_LIMIT", 1200, 1)
	if err != nil {
		return exchangeClientConfig{}, err
	}
	binanceWindow, err := durationEnvStrict("BINANCE_REQUEST_WEIGHT_WINDOW", time.Minute)
	if err != nil {
		return exchangeClientConfig{}, err
	}
	okxLimit, err := intEnvStrict("OKX_MARKET_REQUEST_LIMIT", 20, 1)
	if err != nil {
		return exchangeClientConfig{}, err
	}
	okxWindow, err := durationEnvStrict("OKX_MARKET_REQUEST_WINDOW", 2*time.Second)
	if err != nil {
		return exchangeClientConfig{}, err
	}
	return exchangeClientConfig{
		BinanceBaseURLs:            stringListEnv("BINANCE_BASE_URLS"),
		BinanceRequestWeightLimit:  binanceLimit,
		BinanceRequestWeightWindow: binanceWindow,
		OKXMarketRequestLimit:      okxLimit,
		OKXMarketRequestWindow:     okxWindow,
	}, nil
}

func newCommandFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func durationEnvStrict(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration", key)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", key)
	}
	return parsed, nil
}

func intEnvStrict(key string, fallback int, min int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", key)
	}
	if parsed < min {
		return 0, fmt.Errorf("%s must be greater than or equal to %d", key, min)
	}
	return parsed, nil
}

func boolEnvStrict(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean", key)
	}
}

func safeConfigSummary(pairs ...any) []any {
	values := make([]any, 0, len(pairs))
	for index := 0; index+1 < len(pairs); index += 2 {
		key, ok := pairs[index].(string)
		if !ok || isSensitiveConfigKey(key) {
			continue
		}
		values = append(values, key, pairs[index+1])
	}
	return values
}

func isSensitiveConfigKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, marker := range []string{"database_url", "password", "secret", "token", "api_key", "private_key", "encryption_key", "credential", "dsn"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
