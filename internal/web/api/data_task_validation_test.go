package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestDataSyncTaskRoutesRejectExchangeSymbolMismatch(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		message string
	}{
		{
			name:    "binance hyphen symbol",
			body:    `{"exchange":"binance","symbol":"BTC-USDT","interval":"1m"}`,
			message: "binance symbol must use uppercase compact format such as BTCUSDT",
		},
		{
			name:    "okx compact symbol",
			body:    `{"exchange":"okx","symbol":"BTCUSDT","interval":"1m"}`,
			message: "okx symbol must use uppercase instrument format such as BTC-USDT",
		},
		{
			name:    "unsupported exchange",
			body:    `{"exchange":"kraken","symbol":"BTCUSDT","interval":"1m"}`,
			message: "exchange must be binance or okx",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			repository, server, cookie := newAuthenticatedTestServer(t)

			recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/data/tasks", testCase.body)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			response := decodeAPIError(t, recorder)
			if response.Code != "invalid_request" || response.Message != testCase.message {
				t.Fatalf("unexpected response: %#v", response)
			}
			if len(repository.tasks) != 0 {
				t.Fatalf("invalid task was persisted: %#v", repository.tasks)
			}
		})
	}
}

func TestDataSyncTaskRoutesRejectBlankRequiredText(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"   ","symbol":"BTCUSDT","interval":"1m"}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "invalid_request" || response.Message != "exchange, symbol and interval are required" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if len(repository.tasks) != 0 {
		t.Fatalf("blank task was persisted: %#v", repository.tasks)
	}
}

func TestDataSyncTaskRoutesRejectInvalidIntervalAndWindow(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		message string
	}{
		{
			name:    "unsupported interval",
			body:    `{"exchange":"binance","symbol":"BTCUSDT","interval":"2m"}`,
			message: `unsupported data sync interval "2m"`,
		},
		{
			name: "equal start and end",
			body: `{
				"exchange":"binance",
				"symbol":"BTCUSDT",
				"interval":"1m",
				"startTime":"2026-01-01T00:00:00Z",
				"endTime":"2026-01-01T00:00:00Z"
			}`,
			message: "startTime must be before endTime",
		},
		{
			name: "reversed window",
			body: `{
				"exchange":"binance",
				"symbol":"BTCUSDT",
				"interval":"1m",
				"startTime":"2026-01-01T00:01:00Z",
				"endTime":"2026-01-01T00:00:00Z"
			}`,
			message: "startTime must be before endTime",
		},
		{
			name: "misaligned start",
			body: `{
				"exchange":"binance",
				"symbol":"BTCUSDT",
				"interval":"1m",
				"startTime":"2026-01-01T00:00:30Z",
				"endTime":"2026-01-01T00:01:00Z"
			}`,
			message: "startTime must be aligned to 1m interval",
		},
		{
			name: "misaligned end",
			body: `{
				"exchange":"binance",
				"symbol":"BTCUSDT",
				"interval":"1m",
				"startTime":"2026-01-01T00:00:00Z",
				"endTime":"2026-01-01T00:01:30Z"
			}`,
			message: "endTime must be aligned to 1m interval",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			repository, server, cookie := newAuthenticatedTestServer(t)

			recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/data/tasks", testCase.body)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			response := decodeAPIError(t, recorder)
			if response.Code != "invalid_request" || response.Message != testCase.message {
				t.Fatalf("unexpected response: %#v", response)
			}
			if len(repository.tasks) != 0 {
				t.Fatalf("invalid task was persisted: %#v", repository.tasks)
			}
		})
	}
}

func TestDataSyncTaskRoutesRequireActiveMarketInstrument(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.marketInstruments = append(repository.marketInstruments, marketInstrumentForTest("binance", "SOLUSDT", "inactive"))

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"SOLUSDT","interval":"1m"}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "market_instrument_not_active" ||
		response.Message != "market instrument is inactive in catalog" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if len(repository.tasks) != 0 {
		t.Fatalf("invalid task was persisted: %#v", repository.tasks)
	}
}

func TestDataSyncTaskRoutesCreateActiveMarketInstrument(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.marketInstruments = append(repository.marketInstruments, marketInstrumentForTest("binance", "SOLUSDT", "active"))

	recorder := serveAuthenticated(
		server,
		cookie,
		http.MethodPost,
		"/api/data/tasks",
		`{"exchange":"binance","symbol":"SOLUSDT","interval":"1m"}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(repository.tasks) != 1 || repository.tasks[0].Symbol != "SOLUSDT" {
		t.Fatalf("active task was not persisted: %#v", repository.tasks)
	}
}

func marketInstrumentForTest(exchange string, symbol string, status string) data.MarketInstrument {
	return data.MarketInstrument{
		Exchange:       exchange,
		Symbol:         symbol,
		BaseAsset:      strings.TrimSuffix(symbol, "USDT"),
		QuoteAsset:     "USDT",
		InstrumentType: "spot",
		Status:         status,
		SearchPriority: 20,
	}
}
