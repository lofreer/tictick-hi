package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestCreateTradingTaskDefaultsRiskLimitPct(t *testing.T) {
	repository, server, cookie := newAuthenticatedTestServer(t)

	body := `{
		"name":"Paper EMA",
		"type":"paper",
		"exchange":"binance",
		"accountId":"paper",
		"symbol":"BTCUSDT",
		"interval":"5m",
		"strategyId":"ema-cross",
		"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
		"intentPolicy":{"orderIntent":"execute","notificationChannel":"default"}
	}`

	recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(repository.tradingTasks) != 1 {
		t.Fatalf("trading tasks = %d, want 1", len(repository.tradingTasks))
	}
	if got := repository.tradingTasks[0].IntentPolicy["riskLimitPct"]; got != 10.0 {
		t.Fatalf("riskLimitPct = %#v, want 10.0", got)
	}
}

func TestCreateTradingTaskRejectsInvalidRiskLimitPct(t *testing.T) {
	for _, riskValue := range []string{`101`, `-1`, `"10"`} {
		t.Run(riskValue, func(t *testing.T) {
			repository, server, cookie := newAuthenticatedTestServer(t)
			body := strings.Replace(`{
				"name":"Paper EMA",
				"type":"paper",
				"exchange":"binance",
				"accountId":"paper",
				"symbol":"BTCUSDT",
				"interval":"5m",
				"strategyId":"ema-cross",
				"strategyParams":{"fastPeriod":12,"slowPeriod":26,"orderSize":0.01,"signalMode":"order"},
				"intentPolicy":{"orderIntent":"execute","notificationChannel":"default","riskLimitPct":RISK_VALUE}
			}`, "RISK_VALUE", riskValue, 1)

			recorder := serveAuthenticated(server, cookie, http.MethodPost, "/api/trading/tasks", body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("create status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), "intentPolicy.riskLimitPct must be a number between 0 and 100") {
				t.Fatalf("unexpected response: %s", recorder.Body.String())
			}
			if len(repository.tradingTasks) != 0 {
				t.Fatalf("invalid risk task was persisted: %#v", repository.tradingTasks)
			}
		})
	}
}
