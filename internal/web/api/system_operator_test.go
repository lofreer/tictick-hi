package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestOperatorCannotDisableSelf(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/operators/op_admin/disable", "")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("disable self status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "invalid_state" || response.Message != "current operator cannot be disabled" {
		t.Fatalf("unexpected self disable response: %#v", response)
	}

	operator, err := repository.AuthenticateOperator(t.Context(), testUsername, testPassword)
	if err != nil {
		t.Fatalf("current operator was disabled: %v", err)
	}
	if !operator.Enabled {
		t.Fatalf("current operator enabled = false")
	}
	for _, event := range repository.auditEvents {
		if event.Action == "operator.disable" && event.ResourceID == "op_admin" {
			t.Fatalf("self disable failure was recorded as success audit: %#v", event)
		}
	}
}

func TestCreateOperatorRejectsWeakPassword(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/operators",
		`{"username":"weak","password":"short","enabled":true}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("weak password status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Message != "password must be at least 8 characters" {
		t.Fatalf("unexpected weak password response: %#v", response)
	}
}

func TestRepositoryRejectsDisablingLastEnabledOperator(t *testing.T) {
	repository := newFakeRepository()

	_, err := repository.SetOperatorEnabled(t.Context(), "op_admin", false)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("SetOperatorEnabled error = %v, want invalid state", err)
	}
	if _, err := repository.AuthenticateOperator(t.Context(), testUsername, testPassword); err != nil {
		t.Fatalf("last enabled operator was disabled: %v", err)
	}
}
