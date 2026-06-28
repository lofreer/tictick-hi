package binance

import (
	"context"
	"encoding/json"
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
		if !exchange.IsTemporaryEndpointError(err) {
			allTemporary = false
		}
		errors = append(errors, exchange.EndpointErrorSummary(baseURL, err))
	}
	message := strings.Join(errors, "; ")
	if allTemporary {
		return nil, exchange.NewTemporaryError("binance klines temporary unavailable: "+message, nil)
	}
	return nil, fmt.Errorf("binance klines unavailable: %s", message)
}

func (client *MarketClient) FetchInstruments(ctx context.Context) ([]data.MarketInstrument, error) {
	var errors []string
	allTemporary := true
	for _, baseURL := range client.baseURLs {
		instruments, err := client.fetchInstrumentsFrom(ctx, baseURL)
		if err == nil {
			return instruments, nil
		}
		if !exchange.IsTemporaryEndpointError(err) {
			allTemporary = false
		}
		errors = append(errors, exchange.EndpointErrorSummary(baseURL, err))
	}
	message := strings.Join(errors, "; ")
	if allTemporary {
		return nil, exchange.NewTemporaryError("binance instruments temporary unavailable: "+message, nil)
	}
	return nil, fmt.Errorf("binance instruments unavailable: %s", message)
}

func (client *MarketClient) fetchInstrumentsFrom(
	ctx context.Context,
	baseURL string,
) ([]data.MarketInstrument, error) {
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api/v3/exchangeInfo")
	if err != nil {
		return nil, fmt.Errorf("binance endpoint: %w", err)
	}

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
		return nil, exchange.HTTPStatusError{Code: response.StatusCode, Status: response.Status}
	}

	var envelope binanceExchangeInfo
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode binance instruments: %w", err)
	}

	instruments := make([]data.MarketInstrument, 0, len(envelope.Symbols))
	for _, symbol := range envelope.Symbols {
		instrument, ok := symbol.toInstrument()
		if ok {
			instruments = append(instruments, instrument)
		}
	}
	return instruments, nil
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
		return nil, exchange.HTTPStatusError{Code: response.StatusCode, Status: response.Status}
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

type binanceKline []json.RawMessage

type binanceExchangeInfo struct {
	Symbols []binanceSymbol `json:"symbols"`
}

type binanceSymbol struct {
	Symbol               string `json:"symbol"`
	Status               string `json:"status"`
	BaseAsset            string `json:"baseAsset"`
	QuoteAsset           string `json:"quoteAsset"`
	IsSpotTradingAllowed bool   `json:"isSpotTradingAllowed"`
}

func (symbol binanceSymbol) toInstrument() (data.MarketInstrument, bool) {
	if strings.TrimSpace(symbol.Symbol) == "" ||
		strings.TrimSpace(symbol.BaseAsset) == "" ||
		strings.TrimSpace(symbol.QuoteAsset) == "" ||
		!symbol.IsSpotTradingAllowed {
		return data.MarketInstrument{}, false
	}
	status := "inactive"
	if symbol.Status == "TRADING" {
		status = "active"
	}
	return data.MarketInstrument{
		Exchange:       "binance",
		Symbol:         strings.ToUpper(symbol.Symbol),
		BaseAsset:      strings.ToUpper(symbol.BaseAsset),
		QuoteAsset:     strings.ToUpper(symbol.QuoteAsset),
		InstrumentType: "spot",
		Status:         status,
		SearchPriority: 100,
	}, true
}

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
