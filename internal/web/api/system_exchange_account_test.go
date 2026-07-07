package api

import (
	"net/http"
	"testing"
)

func TestSystemExchangeAccountRejectsBlankFields(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/exchange-accounts",
		`{"exchange":"   ","alias":"main","apiKey":"key","apiSecret":"secret","enabled":true}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("blank exchange status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Message != "exchange, alias, apiKey and apiSecret are required" {
		t.Fatalf("unexpected blank exchange response: %#v", response)
	}
}
