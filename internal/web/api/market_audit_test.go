package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestMarketInstrumentSyncRouteAuditsSuccess(t *testing.T) {
	repository := newFakeRepository()
	server := NewServerWithConfig(repository, Config{
		InstrumentClients: map[string]exchange.InstrumentClient{
			"binance": fakeInstrumentClient{instruments: []data.MarketInstrument{
				{Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
				{Symbol: "DELISTUSDT", BaseAsset: "DELIST", QuoteAsset: "USDT", InstrumentType: "spot", Status: "inactive"},
			}},
		},
	})
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("sync status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "market_instrument.sync", "market_instrument_catalog", "binance")
	if event.Outcome != "success" ||
		event.Metadata["exchange"] != "binance" ||
		event.Metadata["activeCount"] != "1" ||
		event.Metadata["inactiveCount"] != "1" ||
		event.Metadata["pausedDataSyncTaskCount"] != "0" ||
		event.Metadata["restoredDataSyncTaskCount"] != "0" {
		t.Fatalf("unexpected market sync success audit metadata: %#v", event.Metadata)
	}
}

func TestMarketInstrumentSyncRouteAuditsUnavailableClient(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("sync status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "market_instrument.sync", "market_instrument_catalog", "binance")
	if event.Outcome != "failure" ||
		event.Metadata["exchange"] != "binance" ||
		event.Metadata["reason"] != string(apiErrorMarketInstrumentSyncUnavailable) {
		t.Fatalf("unexpected market sync unavailable audit metadata: %#v", event.Metadata)
	}
}

func TestMarketInstrumentSyncRouteRequiresAdmin(t *testing.T) {
	repository := newFakeRepository()
	repository.operators[0].Role = data.OperatorRoleOperator
	server := NewServerWithConfig(repository, Config{
		InstrumentClients: map[string]exchange.InstrumentClient{
			"binance": fakeInstrumentClient{instruments: []data.MarketInstrument{
				{Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
			}},
		},
	})
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("sync status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "forbidden" || response.Message != "admin operator role is required" {
		t.Fatalf("unexpected admin required response: %#v", response)
	}
	if len(repository.marketInstruments) != 1 || repository.marketInstruments[0].Symbol != "BTCUSDT" {
		t.Fatalf("non-admin changed market instruments: %#v", repository.marketInstruments)
	}
	event := assertAuditAction(t, repository.auditEvents, "market_instrument.sync", "market_instrument_catalog", "binance")
	if event.Outcome != "failure" ||
		event.Metadata["reason"] != "admin_required" ||
		event.Metadata["actorRole"] != data.OperatorRoleOperator {
		t.Fatalf("unexpected market sync admin audit metadata: %#v", event.Metadata)
	}
}

func TestMarketInstrumentSyncRouteAuditsFetchFailure(t *testing.T) {
	repository := newFakeRepository()
	server := NewServerWithConfig(repository, Config{
		InstrumentClients: map[string]exchange.InstrumentClient{
			"okx": fakeInstrumentClient{err: errors.New("okx instruments temporary unavailable: www.okx.com: EOF")},
		},
	})
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=okx", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("sync status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	event := assertAuditAction(t, repository.auditEvents, "market_instrument.sync", "market_instrument_catalog", "okx")
	if event.Outcome != "failure" ||
		event.Metadata["exchange"] != "okx" ||
		event.Metadata["reason"] != string(apiErrorMarketInstrumentSyncFailed) {
		t.Fatalf("unexpected market sync fetch failure audit metadata: %#v", event.Metadata)
	}
	for key, value := range event.Metadata {
		if value == "okx instruments temporary unavailable: www.okx.com: EOF" {
			t.Fatalf("market sync fetch failure audit leaked external error in %s=%q", key, value)
		}
	}
}

func TestMarketInstrumentSyncRouteAuditsReplaceFailure(t *testing.T) {
	base := newFakeRepository()
	repository := &marketInstrumentReplaceFailureRepository{
		fakeRepository: base,
		replaceErr:     errors.New("replace market instruments failed"),
	}
	server := NewServerWithConfig(repository, Config{
		InstrumentClients: map[string]exchange.InstrumentClient{
			"binance": fakeInstrumentClient{instruments: []data.MarketInstrument{
				{Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
			}},
		},
	})
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/market/instruments/sync?exchange=binance", "")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("sync status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(base.marketSyncFailures) != 1 {
		t.Fatalf("market sync failures = %#v, want one", base.marketSyncFailures)
	}
	event := assertAuditAction(t, base.auditEvents, "market_instrument.sync", "market_instrument_catalog", "binance")
	if event.Outcome != "failure" ||
		event.Metadata["exchange"] != "binance" ||
		event.Metadata["reason"] != "store_error" {
		t.Fatalf("unexpected market sync replace failure audit metadata: %#v", event.Metadata)
	}
}

type marketInstrumentReplaceFailureRepository struct {
	*fakeRepository
	replaceErr error
}

func (repository *marketInstrumentReplaceFailureRepository) ReplaceMarketInstruments(
	_ context.Context,
	_ string,
	_ []data.MarketInstrument,
	_ time.Time,
) (data.MarketInstrumentSyncResult, error) {
	return data.MarketInstrumentSyncResult{}, repository.replaceErr
}
