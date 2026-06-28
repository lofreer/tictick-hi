package okx

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

const defaultBaseURL = "https://www.okx.com"

const (
	defaultMarketRequestLimit  = 20
	defaultMarketRequestWindow = 2 * time.Second
	marketRequestWeight        = 1
)

type MarketClient struct {
	baseURL     string
	httpClient  *http.Client
	rateLimiter exchange.RateLimiter
}

type MarketClientOptions struct {
	BaseURL     string
	HTTPClient  *http.Client
	RateLimiter exchange.RateLimiter
}

func NewMarketClient(httpClient *http.Client) *MarketClient {
	return NewMarketClientWithOptions(MarketClientOptions{HTTPClient: httpClient})
}

func NewMarketClientForURL(baseURL string, httpClient *http.Client) *MarketClient {
	return NewMarketClientWithOptions(MarketClientOptions{
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	})
}

func NewMarketClientWithOptions(options MarketClientOptions) *MarketClient {
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	rateLimiter := options.RateLimiter
	if rateLimiter == nil {
		rateLimiter = exchange.NewFixedWindowRateLimiter(defaultMarketRequestLimit, defaultMarketRequestWindow)
	}
	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &MarketClient{baseURL: baseURL, httpClient: httpClient, rateLimiter: rateLimiter}
}

func (client *MarketClient) FetchCandles(
	ctx context.Context,
	request exchange.CandleRequest,
) ([]data.Candle, error) {
	endpoint, err := url.Parse(strings.TrimRight(client.baseURL, "/") + "/api/v5/market/history-candles")
	if err != nil {
		return nil, fmt.Errorf("okx endpoint: %w", err)
	}

	values := endpoint.Query()
	values.Set("instId", normalizeSymbol(request.Symbol))
	values.Set("bar", okxInterval(request.Interval))
	values.Set("before", strconv.FormatInt(request.From.UnixMilli(), 10))
	values.Set("after", strconv.FormatInt(request.To.UnixMilli(), 10))
	values.Set("limit", strconv.Itoa(limit(request.Limit, 100)))
	endpoint.RawQuery = values.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	if err := client.wait(ctx); err != nil {
		return nil, err
	}

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, exchange.NewTemporaryError("okx candles temporary unavailable: "+exchange.EndpointErrorSummary(client.baseURL, err), err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		err := exchange.HTTPStatusError{Code: response.StatusCode, Status: response.Status}
		if exchange.IsTemporaryEndpointError(err) {
			return nil, exchange.NewTemporaryError("okx candles temporary unavailable: "+exchange.EndpointErrorSummary(client.baseURL, err), err)
		}
		return nil, fmt.Errorf("okx candles unavailable: %s", exchange.EndpointErrorSummary(client.baseURL, err))
	}

	var envelope okxCandlesResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode okx candles: %w", err)
	}
	if envelope.Code != "0" {
		err := fmt.Errorf("okx candles code %s: %s", envelope.Code, envelope.Message)
		if isTemporaryOKXCode(envelope.Code) {
			return nil, exchange.NewTemporaryError("okx candles temporary unavailable: "+err.Error(), err)
		}
		return nil, err
	}

	candles := make([]data.Candle, 0, len(envelope.Data))
	duration, err := data.IntervalDuration(request.Interval)
	if err != nil {
		return nil, err
	}
	for index := len(envelope.Data) - 1; index >= 0; index-- {
		candle, err := envelope.Data[index].toCandle(request, duration)
		if err != nil {
			return nil, err
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

func (client *MarketClient) FetchInstruments(ctx context.Context) ([]data.MarketInstrument, error) {
	endpoint, err := url.Parse(strings.TrimRight(client.baseURL, "/") + "/api/v5/public/instruments")
	if err != nil {
		return nil, fmt.Errorf("okx endpoint: %w", err)
	}
	values := endpoint.Query()
	values.Set("instType", "SPOT")
	endpoint.RawQuery = values.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	if err := client.wait(ctx); err != nil {
		return nil, err
	}

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, exchange.NewTemporaryError("okx instruments temporary unavailable: "+exchange.EndpointErrorSummary(client.baseURL, err), err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		err := exchange.HTTPStatusError{Code: response.StatusCode, Status: response.Status}
		if exchange.IsTemporaryEndpointError(err) {
			return nil, exchange.NewTemporaryError("okx instruments temporary unavailable: "+exchange.EndpointErrorSummary(client.baseURL, err), err)
		}
		return nil, fmt.Errorf("okx instruments unavailable: %s", exchange.EndpointErrorSummary(client.baseURL, err))
	}

	var envelope okxInstrumentsResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode okx instruments: %w", err)
	}
	if envelope.Code != "0" {
		err := fmt.Errorf("okx instruments code %s: %s", envelope.Code, envelope.Message)
		if isTemporaryOKXCode(envelope.Code) {
			return nil, exchange.NewTemporaryError("okx instruments temporary unavailable: "+err.Error(), err)
		}
		return nil, err
	}

	instruments := make([]data.MarketInstrument, 0, len(envelope.Data))
	for _, item := range envelope.Data {
		instrument, ok := item.toInstrument()
		if ok {
			instruments = append(instruments, instrument)
		}
	}
	return instruments, nil
}

type okxCandlesResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"msg"`
	Data    []okxCandleRow `json:"data"`
}

type okxInstrumentsResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"msg"`
	Data    []okxInstrument `json:"data"`
}

type okxInstrument struct {
	InstrumentID  string `json:"instId"`
	BaseCurrency  string `json:"baseCcy"`
	QuoteCurrency string `json:"quoteCcy"`
	State         string `json:"state"`
}

func (instrument okxInstrument) toInstrument() (data.MarketInstrument, bool) {
	if strings.TrimSpace(instrument.InstrumentID) == "" ||
		strings.TrimSpace(instrument.BaseCurrency) == "" ||
		strings.TrimSpace(instrument.QuoteCurrency) == "" {
		return data.MarketInstrument{}, false
	}
	status := "inactive"
	if instrument.State == "live" {
		status = "active"
	}
	exchangeStatus := strings.ToLower(strings.TrimSpace(instrument.State))
	if exchangeStatus == "" {
		exchangeStatus = status
	}
	return data.MarketInstrument{
		Exchange:       "okx",
		Symbol:         strings.ToUpper(instrument.InstrumentID),
		BaseAsset:      strings.ToUpper(instrument.BaseCurrency),
		QuoteAsset:     strings.ToUpper(instrument.QuoteCurrency),
		InstrumentType: "spot",
		Status:         status,
		ExchangeStatus: exchangeStatus,
		SearchPriority: 100,
	}, true
}

type okxCandleRow []string

func (row okxCandleRow) toCandle(
	request exchange.CandleRequest,
	duration time.Duration,
) (data.Candle, error) {
	if len(row) < 6 {
		return data.Candle{}, fmt.Errorf("okx candle has %d fields", len(row))
	}

	openMillis, err := strconv.ParseInt(row[0], 10, 64)
	if err != nil {
		return data.Candle{}, fmt.Errorf("okx timestamp: %w", err)
	}
	openTime := time.UnixMilli(openMillis).UTC()

	return data.Candle{
		Exchange:  request.Exchange,
		Symbol:    request.Symbol,
		Interval:  request.Interval,
		OpenTime:  openTime,
		CloseTime: openTime.Add(duration),
		Open:      row[1],
		High:      row[2],
		Low:       row[3],
		Close:     row[4],
		Volume:    row[5],
		IsClosed:  len(row) < 9 || row[8] == "1",
	}, nil
}

func normalizeSymbol(symbol string) string {
	if strings.Contains(symbol, "-") {
		return symbol
	}
	if strings.HasSuffix(symbol, "USDT") {
		return strings.TrimSuffix(symbol, "USDT") + "-USDT"
	}
	return symbol
}

func okxInterval(interval string) string {
	if strings.HasSuffix(interval, "h") || strings.HasSuffix(interval, "d") {
		return strings.ToUpper(interval)
	}
	return interval
}

func isTemporaryOKXCode(code string) bool {
	return code == "50011"
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

func (client *MarketClient) wait(ctx context.Context) error {
	if client.rateLimiter == nil {
		return nil
	}
	if err := client.rateLimiter.Wait(ctx, marketRequestWeight); err != nil {
		return fmt.Errorf("okx rate limit wait: %w", err)
	}
	return nil
}
