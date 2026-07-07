package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
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

func TestSystemExchangeAccountCreateStoreFailureAuditedWithoutSecrets(t *testing.T) {
	base := newFakeRepository()
	repository := &exchangeAccountFailureRepository{
		fakeRepository: base,
		createErr:      errors.New("create failed"),
	}
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/exchange-accounts",
		`{"exchange":"binance","alias":"main","apiKey":"exchange-key-secret","apiSecret":"exchange-api-secret","enabled":true}`,
	)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("create failure status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "internal_error" || response.Message != "internal server error" {
		t.Fatalf("unexpected create failure response: %#v", response)
	}

	event := assertAuditAction(t, base.auditEvents, "exchange_account.create", "exchange_account", "")
	if event.Outcome != "failure" ||
		event.Metadata["reason"] != "store_error" ||
		event.Metadata["exchange"] != "binance" ||
		event.Metadata["alias"] != "main" ||
		event.Metadata["enabled"] != "true" {
		t.Fatalf("unexpected exchange account failure audit metadata: %#v", event.Metadata)
	}
	if _, exists := event.Metadata["apiKey"]; exists {
		t.Fatalf("exchange account failure audit metadata includes apiKey: %#v", event.Metadata)
	}
	if _, exists := event.Metadata["apiSecret"]; exists {
		t.Fatalf("exchange account failure audit metadata includes apiSecret: %#v", event.Metadata)
	}
	for key, value := range event.Metadata {
		if strings.Contains(value, "exchange-key-secret") || strings.Contains(value, "exchange-api-secret") {
			t.Fatalf("exchange account failure audit metadata leaked secret in %s=%q", key, value)
		}
	}
}

type exchangeAccountFailureRepository struct {
	*fakeRepository
	createErr error
}

func (repository *exchangeAccountFailureRepository) CreateExchangeAccount(
	ctx context.Context,
	request data.CreateExchangeAccount,
) (data.ExchangeAccount, error) {
	if repository.createErr != nil {
		return data.ExchangeAccount{}, repository.createErr
	}
	return repository.fakeRepository.CreateExchangeAccount(ctx, request)
}
