package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestOperatorSessionStorePersistsClientContext(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("session_context"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	now := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)
	session := data.OperatorSession{
		ID:         integrationID("os"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("token"),
		RemoteAddr: "203.0.113.24",
		UserAgent:  "tictick-hi-test/1.0",
		ExpiresAt:  now.Add(time.Hour),
	}
	if err := store.CreateOperatorSession(ctx, session); err != nil {
		t.Fatal(err)
	}

	sessions, err := store.ListOperatorSessions(ctx, operator.ID, session.TokenHash, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %#v", sessions)
	}
	if sessions[0].RemoteAddr != session.RemoteAddr || sessions[0].UserAgent != session.UserAgent {
		t.Fatalf("unexpected session context: %#v", sessions[0])
	}
}

func TestOperatorSessionStoreRejectsRevokingCurrentSession(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("session_current"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	session := data.OperatorSession{
		ID:         integrationID("os_current"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("token_current"),
		ExpiresAt:  time.Now().UTC().Add(time.Hour),
	}
	if err := store.CreateOperatorSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	err = store.DeleteOperatorSessionByID(ctx, operator.ID, session.ID, session.TokenHash)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("DeleteOperatorSessionByID error = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeAuthCurrentSessionRevokeForbidden {
		t.Fatalf("DeleteOperatorSessionByID code = %q, %t; want %q, true", code, ok, data.ErrorCodeAuthCurrentSessionRevokeForbidden)
	}
}
