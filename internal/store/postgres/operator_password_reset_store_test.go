package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestOperatorStoreResetsOperatorPasswordAndRevokesSessions(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("password_reset"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	firstSession := data.OperatorSession{
		ID:         integrationID("reset_session_1"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("reset_hash_1"),
		ExpiresAt:  now.Add(time.Hour),
	}
	secondSession := data.OperatorSession{
		ID:         integrationID("reset_session_2"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("reset_hash_2"),
		ExpiresAt:  now.Add(time.Hour),
	}
	if err := store.CreateOperatorSession(ctx, firstSession); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateOperatorSession(ctx, secondSession); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	revokedCount, err := store.ResetOperatorPassword(ctx, operator.ID, "reset456B")
	if err != nil {
		t.Fatal(err)
	}
	if revokedCount != 2 {
		t.Fatalf("revoked session count = %d, want 2", revokedCount)
	}
	if _, err := store.AuthenticateOperator(ctx, operator.Username, "secret123A"); !errors.Is(err, data.ErrUnauthorized) {
		t.Fatalf("old password authentication error = %v, want unauthorized", err)
	}
	if _, err := store.AuthenticateOperator(ctx, operator.Username, "reset456B"); err != nil {
		t.Fatalf("new password authentication failed: %v", err)
	}
	sessions, err := store.ListOperatorSessions(ctx, operator.ID, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions after password reset: %#v", sessions)
	}

	_, err = store.ResetOperatorPassword(ctx, operator.ID, "secret123A")
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeOperatorPasswordReused {
		t.Fatalf("reset to historical password code = %q, %t; want %q, true", code, ok, data.ErrorCodeOperatorPasswordReused)
	}
}
