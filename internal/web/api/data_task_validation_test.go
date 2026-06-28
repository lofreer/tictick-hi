package api

import (
	"net/http"
	"testing"
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
