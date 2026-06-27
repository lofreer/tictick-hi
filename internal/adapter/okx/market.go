package okx

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

const defaultBaseURL = "https://www.okx.com"

type MarketClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewMarketClient(httpClient *http.Client) *MarketClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &MarketClient{baseURL: defaultBaseURL, httpClient: httpClient}
}

func NewMarketClientForURL(baseURL string, httpClient *http.Client) *MarketClient {
	client := NewMarketClient(httpClient)
	client.baseURL = baseURL
	return client
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

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, exchange.NewTemporaryError("okx candles temporary unavailable: "+endpointError(client.baseURL, err), err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		err := statusError{code: response.StatusCode, status: response.Status}
		if isTemporaryEndpointError(err) {
			return nil, exchange.NewTemporaryError("okx candles temporary unavailable: "+endpointError(client.baseURL, err), err)
		}
		return nil, fmt.Errorf("okx candles unavailable: %s", endpointError(client.baseURL, err))
	}

	var envelope okxCandlesResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode okx candles: %w", err)
	}
	if envelope.Code != "0" {
		return nil, fmt.Errorf("okx candles code %s: %s", envelope.Code, envelope.Message)
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

type okxCandlesResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"msg"`
	Data    []okxCandleRow `json:"data"`
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

func limit(value int, max int) int {
	if value <= 0 {
		return max
	}
	if value > max {
		return max
	}
	return value
}

type statusError struct {
	code   int
	status string
}

func (err statusError) Error() string {
	return "status " + err.status
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

func isTemporaryEndpointError(err error) bool {
	var statusErr statusError
	if errors.As(err, &statusErr) {
		return statusErr.code == http.StatusTooManyRequests || statusErr.code >= 500
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr) || errors.Is(err, context.DeadlineExceeded)
}
