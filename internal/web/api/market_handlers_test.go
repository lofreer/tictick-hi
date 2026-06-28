package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
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

func TestMarketInstrumentSyncRouteRefreshesCatalog(t *testing.T) {
	repository := newFakeRepository()
	repository.marketInstruments = []data.MarketInstrument{
		{Exchange: "binance", Symbol: "OLDUSDT", BaseAsset: "OLD", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
	}
	server := NewServerWithConfig(repository, Config{
		InstrumentClients: map[string]exchange.InstrumentClient{
			"binance": fakeInstrumentClient{instruments: []data.MarketInstrument{
				{Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
				{Symbol: "DELISTUSDT", BaseAsset: "DELIST", QuoteAsset: "USDT", InstrumentType: "spot", Status: "inactive"},
			}},
		},
	})
	auth := loginTestOperator(t, server)

	missingCSRF := serveAuthenticatedWithoutCSRF(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing csrf status = %d body = %s", missingCSRF.Code, missingCSRF.Body.String())
	}

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("sync status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.MarketInstrumentSyncResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Exchange != "binance" || result.ActiveCount != 1 || result.InactiveCount != 1 {
		t.Fatalf("unexpected sync result: %#v", result)
	}

	listRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/market/instruments?exchange=binance&q=sol", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var instruments []data.MarketInstrument
	if err := json.NewDecoder(listRecorder.Body).Decode(&instruments); err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 1 || instruments[0].Symbol != "SOLUSDT" {
		t.Fatalf("instruments = %#v, want SOLUSDT only", instruments)
	}
}

func TestMarketInstrumentSyncRouteRejectsUnavailableClient(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "request_failed" {
		t.Fatalf("unexpected error response: %#v", response)
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

type fakeInstrumentClient struct {
	instruments []data.MarketInstrument
}

func (client fakeInstrumentClient) FetchInstruments(context.Context) ([]data.MarketInstrument, error) {
	return append([]data.MarketInstrument(nil), client.instruments...), nil
}
