package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestMarketInstrumentRoutesRequireAuthentication(t *testing.T) {
	server := NewServer(newFakeRepository(), "")

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/market/instruments?exchange=binance", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "unauthorized" {
		t.Fatalf("unexpected auth response: %#v", response)
	}
}

func TestMarketInstrumentRoutesSearchCatalog(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.marketInstruments = []data.MarketInstrument{
		{Exchange: "binance", Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
		{Exchange: "binance", Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
		{Exchange: "binance", Symbol: "SOLBTC", BaseAsset: "SOL", QuoteAsset: "BTC", InstrumentType: "spot", Status: "inactive"},
		{Exchange: "okx", Symbol: "SOL-USDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
	}

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/market/instruments?exchange=binance&q=sol&limit=100", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var instruments []data.MarketInstrument
	if err := json.NewDecoder(recorder.Body).Decode(&instruments); err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 1 || instruments[0].Symbol != "SOLUSDT" {
		t.Fatalf("instruments = %#v, want SOLUSDT only", instruments)
	}
}

func TestMarketInstrumentRoutesRejectInvalidQuery(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	cases := []string{
		"/api/market/instruments",
		"/api/market/instruments?exchange=coinbase",
		"/api/market/instruments?exchange=binance&limit=zero",
		"/api/market/instruments?exchange=binance&limit=0",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			recorder := serveAuthenticated(server, auth, http.MethodGet, path, "")
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			response := decodeAPIError(t, recorder)
			if response.Code != "invalid_request" {
				t.Fatalf("unexpected error response: %#v", response)
			}
		})
	}
}
