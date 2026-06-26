package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

const defaultBaseURL = "https://api.binance.com"

type MarketClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewMarketClient(httpClient *http.Client) *MarketClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
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
	endpoint, err := url.Parse(client.baseURL + "/api/v3/klines")
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
		return nil, fmt.Errorf("binance klines: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("binance klines status: %s", response.Status)
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
