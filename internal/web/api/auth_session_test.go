package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestAuthSessionManagementRoutes(t *testing.T) {
	server := NewServer(newFakeRepository(), "")
	currentAuth := loginTestOperator(t, server)
	otherAuth := loginTestOperator(t, server)

	listRecorder := serveAuthenticated(server, currentAuth, http.MethodGet, "/api/auth/sessions", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var sessions []data.OperatorSession
	if err := json.NewDecoder(listRecorder.Body).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("session count = %d sessions = %#v", len(sessions), sessions)
	}
	currentID := ""
	otherID := ""
	for _, session := range sessions {
		if session.ID == "" {
			t.Fatalf("session id was empty: %#v", session)
		}
		if session.TokenHash != "" {
			t.Fatalf("session token hash leaked: %#v", session)
		}
		if session.Current {
			currentID = session.ID
		} else {
			otherID = session.ID
		}
	}
	if currentID == "" || otherID == "" {
		t.Fatalf("expected current and non-current sessions: %#v", sessions)
	}

	missingCSRFRecorder := serveAuthenticatedWithoutCSRF(
		server,
		currentAuth,
		http.MethodDelete,
		"/api/auth/sessions/"+otherID,
		"",
	)
	if missingCSRFRecorder.Code != http.StatusForbidden {
		t.Fatalf("missing csrf delete status = %d body = %s", missingCSRFRecorder.Code, missingCSRFRecorder.Body.String())
	}

	currentDeleteRecorder := serveAuthenticated(server, currentAuth, http.MethodDelete, "/api/auth/sessions/"+currentID, "")
	if currentDeleteRecorder.Code != http.StatusConflict {
		t.Fatalf("delete current session status = %d body = %s", currentDeleteRecorder.Code, currentDeleteRecorder.Body.String())
	}
	currentDeleteResponse := decodeAPIError(t, currentDeleteRecorder)
	if currentDeleteResponse.Code != "invalid_state" {
		t.Fatalf("unexpected delete current response: %#v", currentDeleteResponse)
	}

	deleteRecorder := serveAuthenticated(server, currentAuth, http.MethodDelete, "/api/auth/sessions/"+otherID, "")
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete other session status = %d body = %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}

	revokedRecorder := serveAuthenticated(server, otherAuth, http.MethodGet, "/api/auth/me", "")
	if revokedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status = %d body = %s", revokedRecorder.Code, revokedRecorder.Body.String())
	}

	remainingRecorder := serveAuthenticated(server, currentAuth, http.MethodGet, "/api/auth/sessions", "")
	if remainingRecorder.Code != http.StatusOK {
		t.Fatalf("remaining sessions status = %d body = %s", remainingRecorder.Code, remainingRecorder.Body.String())
	}
	sessions = nil
	if err := json.NewDecoder(remainingRecorder.Body).Decode(&sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || !sessions[0].Current || sessions[0].ID != currentID {
		t.Fatalf("unexpected remaining sessions: %#v", sessions)
	}
}
