package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestCreateLiveTradingTaskRequiresConfirmation(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)
	repository.accounts = append(repository.accounts, data.ExchangeAccount{
		ID:               "acct_live",
		Exchange:         "binance",
		Alias:            "main",
		Enabled:          true,
		CredentialStatus: "encrypted",
	})

	body := `{
		"name":"Live EMA",
		"type":"live",
		"exchange":"binance",
		"accountId":"acct_live",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"notify","notificationChannel":"default"}
	}`

	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", body)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing confirmation status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "liveConfirmation must be LIVE for live tasks") {
		t.Fatalf("unexpected missing confirmation body: %s", recorder.Body.String())
	}
	if len(repository.tradingTasks) != 0 {
		t.Fatalf("live task without confirmation was persisted: %#v", repository.tradingTasks)
	}

	confirmedBody := strings.Replace(body, `"type":"live",`, `"type":"live","liveConfirmation":"LIVE",`, 1)
	confirmedRecorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", confirmedBody)
	if confirmedRecorder.Code != http.StatusCreated {
		t.Fatalf("confirmed live status = %d body = %s", confirmedRecorder.Code, confirmedRecorder.Body.String())
	}
}
