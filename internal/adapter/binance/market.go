package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

const defaultBaseURL = "https://api.binance.com"

var defaultBaseURLs = []string{
	defaultBaseURL,
	"https://data-api.binance.vision",
}

type MarketClient struct {
	baseURLs   []string
	httpClient *http.Client
}

func NewMarketClient(httpClient *http.Client) *MarketClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &MarketClient{baseURLs: append([]string(nil), defaultBaseURLs...), httpClient: httpClient}
}

func NewMarketClientWithBaseURLs(baseURLs []string, httpClient *http.Client) *MarketClient {
	client := NewMarketClient(httpClient)
	client.baseURLs = normalizeBaseURLs(baseURLs)
	return client
}

func NewMarketClientForURL(baseURL string, httpClient *http.Client) *MarketClient {
	return NewMarketClientWithBaseURLs([]string{baseURL}, httpClient)
}

func (client *MarketClient) FetchCandles(
	ctx context.Context,
	request exchange.CandleRequest,
) ([]data.Candle, error) {
	var errors []string
	allTemporary := true
	for _, baseURL := range client.baseURLs {
		candles, err := client.fetchCandlesFrom(ctx, baseURL, request)
		if err == nil {
			return candles, nil
		}
		if !isTemporaryEndpointError(err) {
			allTemporary = false
		}
		errors = append(errors, endpointError(baseURL, err))
	}
	message := strings.Join(errors, "; ")
	if allTemporary {
		return nil, exchange.NewTemporaryError("binance klines temporary unavailable: "+message, nil)
	}
	return nil, fmt.Errorf("binance klines unavailable: %s", message)
}

func (client *MarketClient) fetchCandlesFrom(
	ctx context.Context,
	baseURL string,
	request exchange.CandleRequest,
) ([]data.Candle, error) {
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api/v3/klines")
	if err != nil {
		return nil, fmt.Errorf("binance endpoint: %w", err)
	}

	values := endpoint.Query()
	values.Set("symbol", request.Symbol)
	values.Set("interval", request.Interval)
	values.Set("startTime", strconv.FormatInt(request.From.UnixMilli(), 10))
	values.Set("endTime", strconv.FormatInt(request.To.UnixMilli(), 10))
	values.Set("limit", strconv.Itoa(limit(request.Limit, 1000)))
	endpoint.RawQuery = values.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, statusError{code: response.StatusCode, status: response.Status}
	}

	var rows []binanceKline
	if err := json.NewDecoder(response.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode binance klines: %w", err)
	}

	candles := make([]data.Candle, 0, len(rows))
	now := time.Now().UTC()
	for _, row := range rows {
		candle, err := row.toCandle(request, now)
		if err != nil {
			return nil, err
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

func normalizeBaseURLs(baseURLs []string) []string {
	normalized := make([]string, 0, len(baseURLs))
	for _, baseURL := range baseURLs {
		baseURL = strings.TrimSpace(baseURL)
		if baseURL != "" {
			normalized = append(normalized, baseURL)
		}
	}
	if len(normalized) == 0 {
		return append([]string(nil), defaultBaseURLs...)
	}
	return normalized
}

func endpointError(baseURL string, err error) string {
	parsed, parseErr := url.Parse(baseURL)
	host := baseURL
	if parseErr == nil && parsed.Host != "" {
		host = parsed.Host
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Sprintf("%s: %s %v", host, urlErr.Op, urlErr.Err)
	}
	return fmt.Sprintf("%s: %v", host, err)
}

type statusError struct {
	code   int
	status string
}

func (err statusError) Error() string {
	return "status " + err.status
}

func isTemporaryEndpointError(err error) bool {
	var statusErr statusError
	if errors.As(err, &statusErr) {
		return statusErr.code == http.StatusTooManyRequests || statusErr.code >= 500
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr) || errors.Is(err, context.DeadlineExceeded)
}

type binanceKline []json.RawMessage

func (row binanceKline) toCandle(request exchange.CandleRequest, now time.Time) (data.Candle, error) {
	if len(row) < 7 {
		return data.Candle{}, fmt.Errorf("binance kline has %d fields", len(row))
	}

	openMillis, err := decodeInt64(row[0])
	if err != nil {
		return data.Candle{}, err
	}
	closeMillis, err := decodeInt64(row[6])
	if err != nil {
		return data.Candle{}, err
	}

	closeTime := time.UnixMilli(closeMillis + 1).UTC()
	return data.Candle{
		Exchange:  request.Exchange,
		Symbol:    request.Symbol,
		Interval:  request.Interval,
		OpenTime:  time.UnixMilli(openMillis).UTC(),
		CloseTime: closeTime,
		Open:      decodeString(row[1]),
		High:      decodeString(row[2]),
		Low:       decodeString(row[3]),
		Close:     decodeString(row[4]),
		Volume:    decodeString(row[5]),
		IsClosed:  !closeTime.After(now),
	}, nil
}

func decodeInt64(raw json.RawMessage) (int64, error) {
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("decode int64: %w", err)
	}
	return value, nil
}

func decodeString(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func limit(value int, max int) int {
	if value <= 0 {
		return max
	}
	if value > max {
		return max
	}
	return value
}
